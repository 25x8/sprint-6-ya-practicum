package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	config "github.com/25x8/sprint-6-ya-practicum/internal"
	"github.com/25x8/sprint-6-ya-practicum/internal/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/db"
	"github.com/gorilla/mux"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Warning: could not load config file: %v", err)
	}

	// Парсим флаги командной строки
	var (
		runAddr     string
		databaseURI string
	)

	flag.StringVar(&runAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&databaseURI, "d", "", "database URI")
	flag.Parse()

	// Проверяем переменные окружения
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		log.Printf("Using RUN_ADDRESS from environment: %s", envRunAddr)
		runAddr = envRunAddr
	} else if envRunAddrAccrual := os.Getenv("RUN_ADDRESS_ACCRUAL"); envRunAddrAccrual != "" {
		log.Printf("Using RUN_ADDRESS_ACCRUAL from environment: %s", envRunAddrAccrual)
		runAddr = envRunAddrAccrual
	}

	if envDatabaseURI := os.Getenv("DATABASE_URI"); envDatabaseURI != "" {
		log.Printf("Using DATABASE_URI from environment")
		databaseURI = envDatabaseURI
	} else if databaseURI == "" && cfg != nil {
		// Если URI базы данных не указан ни через флаг, ни через переменную окружения,
		// используем значение из конфигурации
		databaseURI = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.AccrualDBUser,
			cfg.AccrualDBPassword,
			cfg.AccrualDBHost,
			cfg.AccrualDBPort,
			cfg.AccrualDBName,
			cfg.AccrualDBSSLMode)
		log.Printf("Using database URI from config")
	}

	log.Printf("Final runAddr value: %s", runAddr)

	// Убедимся, что адрес имеет правильный формат
	if !strings.Contains(runAddr, ":") {
		log.Printf("Warning: runAddr does not contain port separator ':', adding default port (:8080)")
		runAddr = runAddr + ":8080"
	}

	// Если адрес содержит только порт (например, :8080), добавим localhost
	if runAddr[0] == ':' {
		log.Printf("Warning: runAddr starts with ':', assuming localhost")
		runAddr = "localhost" + runAddr
	}

	log.Printf("Final server address: %s", runAddr)

	// Проверяем обязательные параметры
	if databaseURI == "" {
		log.Fatal("Database URI is required")
	}

	// Применяем миграции
	if err := db.ApplyMigrations(databaseURI); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	// Инициализируем базу данных
	database, err := db.InitDB(databaseURI)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.CloseDB()

	repo := accrual.NewPostgresRepository(database)
	handler := accrual.NewHandler(repo)

	// Обеспечиваем корректное завершение работы пула воркеров
	defer handler.Shutdown()

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
