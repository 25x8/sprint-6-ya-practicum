package repository

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// DB представляет соединение с базой данных
var DB *sql.DB

// InitDB инициализирует соединение с базой данных
func InitDB(dsn string) (*sql.DB, error) {
	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение
	if err := DB.Ping(); err != nil {
		return nil, err
	}

	// Применяем миграции
	if err := ApplyMigrations(DB); err != nil {
		return nil, err
	}

	log.Println("Database connection established")
	return DB, nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}
