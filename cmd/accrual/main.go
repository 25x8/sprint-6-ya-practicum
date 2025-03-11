package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/25x8/sprint-6-ya-practicum/internal/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/db"
	"github.com/gorilla/mux"
)

func main() {
	if err := db.ApplyMigrations(); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	database, err := db.InitDB()
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Канал для получения сигналов завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Starting server on %s", addr)
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
