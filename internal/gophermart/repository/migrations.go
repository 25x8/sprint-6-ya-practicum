package repository

import (
	"database/sql"
	"log"
)

// Migrations содержит SQL-запросы для создания таблиц
var Migrations = []string{
	// Создание таблицы пользователей
	`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		balance NUMERIC(10, 2) NOT NULL DEFAULT 0,
		withdrawn NUMERIC(10, 2) NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL
	)`,

	// Создание таблицы заказов
	`CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		number VARCHAR(255) NOT NULL UNIQUE,
		status VARCHAR(20) NOT NULL,
		accrual NUMERIC(10, 2) NOT NULL DEFAULT 0,
		uploaded_at TIMESTAMP NOT NULL
	)`,

	// Создание таблицы операций списания
	`CREATE TABLE IF NOT EXISTS withdrawals (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		order_number VARCHAR(255) NOT NULL,
		sum NUMERIC(10, 2) NOT NULL,
		processed_at TIMESTAMP NOT NULL
	)`,
}

// ApplyMigrations применяет миграции к базе данных
func ApplyMigrations(db *sql.DB) error {
	for i, migration := range Migrations {
		_, err := db.Exec(migration)
		if err != nil {
			log.Printf("Failed to apply migration %d: %v", i, err)
			return err
		}
	}
	return nil
}
