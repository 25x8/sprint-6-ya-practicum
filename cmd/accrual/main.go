package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Starting server on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}

}
