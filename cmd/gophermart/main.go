package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	config "github.com/25x8/sprint-6-ya-practicum/internal"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/handlers"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
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
		log.Printf("Using RUN_ADDRESS from environment: %s", envRunAddr)
		runAddr = envRunAddr
	}

	if envDatabaseURI := os.Getenv("DATABASE_URI"); envDatabaseURI != "" {
		log.Printf("Using DATABASE_URI from environment")
		databaseURI = envDatabaseURI
	} else if databaseURI == "" && cfg != nil {
		// Если URI базы данных не указан ни через флаг, ни через переменную окружения,
		// используем значение из конфигурации
		databaseURI = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.GophermartDBUser,
			cfg.GophermartDBPassword,
			cfg.GophermartDBHost,
			cfg.GophermartDBPort,
			cfg.GophermartDBName,
			cfg.GophermartDBSSLMode)
		log.Printf("Using database URI from config")
	}

	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		log.Printf("Using ACCRUAL_SYSTEM_ADDRESS from environment: %s", envAccrualAddr)
		accrualAddr = envAccrualAddr
	} else if accrualAddr == "" && cfg != nil {
		// Если адрес accrual системы не указан ни через флаг, ни через переменную окружения,
		// используем значение из конфигурации
		accrualAddr = cfg.AccrualSystemAddress
		log.Printf("Using accrual system address from config: %s", accrualAddr)
	}

	log.Printf("Using runAddr: %s", runAddr)
	log.Printf("Using accrualAddr: %s", accrualAddr)

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

	// Создаем логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем клиент для системы accrual
	accrualClient := accrual.NewClient(accrualAddr)

	// Создаем обработчики
	handler := handlers.NewHandler(repo, authService, accrualClient, logger)

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
