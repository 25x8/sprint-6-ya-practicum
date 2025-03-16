package repository

import (
	"database/sql"
	"fmt"
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
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`,

	// Создание таблицы заказов
	`CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		number VARCHAR(255) NOT NULL UNIQUE,
		status VARCHAR(20) NOT NULL,
		accrual NUMERIC(10, 2) NOT NULL DEFAULT 0,
		uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`,

	// Создание таблицы операций списания
	`CREATE TABLE IF NOT EXISTS withdrawals (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		order_number VARCHAR(255) NOT NULL,
		sum NUMERIC(10, 2) NOT NULL,
		processed_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`,
}

// AlterMigrations содержит SQL-запросы для обновления таблиц
var AlterMigrations = []string{
	// Переименование колонки password в password_hash, если такая колонка существует
	`DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'password'
		) THEN
			ALTER TABLE users RENAME COLUMN password TO password_hash;
		END IF;
	END $$;`,

	// Добавление колонки balance, если она отсутствует
	`DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'balance'
		) THEN
			ALTER TABLE users ADD COLUMN balance NUMERIC(10, 2) NOT NULL DEFAULT 0;
		END IF;
	END $$;`,

	// Добавление колонки withdrawn, если она отсутствует
	`DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'withdrawn'
		) THEN
			ALTER TABLE users ADD COLUMN withdrawn NUMERIC(10, 2) NOT NULL DEFAULT 0;
		END IF;
	END $$;`,
}

// ApplyMigrations применяет миграции к базе данных
func ApplyMigrations(db *sql.DB) error {
	// Применяем основные миграции из массива
	for i, migration := range Migrations {
		_, err := db.Exec(migration)
		if err != nil {
			return fmt.Errorf("failed to apply migration #%d: %w", i+1, err)
		}
	}

	// Применяем миграции для изменения таблиц
	for i, migration := range AlterMigrations {
		_, err := db.Exec(migration)
		if err != nil {
			return fmt.Errorf("failed to apply alter migration #%d: %w", i+1, err)
		}
	}

	return nil
}
