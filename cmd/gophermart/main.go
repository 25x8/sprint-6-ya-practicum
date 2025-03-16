package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/handlers"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
	"github.com/gorilla/mux"
)

func main() {
	// Парсим флаги командной строки
	var (
		runAddr        string
		databaseURI    string
		accrualAddr    string
		signingKey     string
		tokenTTL       time.Duration
		accrualTimeout time.Duration
	)

	flag.StringVar(&runAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&databaseURI, "d", "", "database URI")
	flag.StringVar(&accrualAddr, "r", "", "accrual system address")
	flag.StringVar(&signingKey, "s", "secret", "JWT signing key")
	flag.DurationVar(&tokenTTL, "t", 24*time.Hour, "JWT token TTL")
	flag.DurationVar(&accrualTimeout, "timeout", 5*time.Second, "accrual system request timeout")
	flag.Parse()

	// Проверяем переменные окружения
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		runAddr = envRunAddr
	}
	if envDatabaseURI := os.Getenv("DATABASE_URI"); envDatabaseURI != "" {
		databaseURI = envDatabaseURI
	}
	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		accrualAddr = envAccrualAddr
	}

	// Проверяем обязательные параметры
	if databaseURI == "" {
		log.Fatal("Database URI is required")
	}
	if accrualAddr == "" {
		log.Fatal("Accrual system address is required")
	}

	// Инициализируем базу данных
	db, err := repository.InitDB(databaseURI)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repository.CloseDB()

	// Создаем репозиторий
	repo := repository.NewPostgresRepository(db)

	// Создаем сервис аутентификации
	authService := auth.NewAuth(repo, signingKey, tokenTTL)

	// Создаем клиент для системы accrual
	accrualClient := accrual.NewClient(accrualAddr, accrualTimeout)

	// Создаем обработчики
	handler := handlers.NewHandler(repo, authService, accrualClient)

	// Создаем маршрутизатор
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:    runAddr,
		Handler: router,
	}

	// Канал для получения сигналов завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Starting server on %s", runAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	<-stop
	log.Println("Shutting down server...")

	// Создаем контекст с таймаутом для завершения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Корректно завершаем работу сервера
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}
