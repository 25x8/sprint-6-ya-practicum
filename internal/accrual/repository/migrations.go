package repository

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// ApplyMigrations применяет миграции для сервиса accrual
func ApplyMigrations(db *sql.DB) error {
	// Пытаемся обновить ограничение CHECK для поля status в таблице orders,
	// чтобы оно включало значение 'NEW'
	_, err := db.Exec(`
		DO $$
		BEGIN
			ALTER TABLE IF EXISTS orders 
			DROP CONSTRAINT IF EXISTS orders_status_check;
			
			ALTER TABLE IF EXISTS orders 
			ADD CONSTRAINT orders_status_check 
			CHECK (status IN ('NEW', 'REGISTERED', 'INVALID', 'PROCESSING', 'PROCESSED'));
		EXCEPTION WHEN others THEN
			-- Игнорируем ошибки, если таблица еще не существует
			NULL;
		END $$;
	`)
	if err != nil {
		log.Printf("Warning: could not update CHECK constraint: %v", err)
	}

	// Ищем директорию с миграциями
	migrationsDirs := []string{
		"migrations/accrual",
		"../migrations/accrual",
		"../../migrations/accrual",
	}

	var migrationsPath string
	for _, dir := range migrationsDirs {
		if _, err := os.Stat(dir); err == nil {
			absPath, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
			}
			migrationsPath = fmt.Sprintf("file://%s", absPath)
			log.Printf("Using migrations from: %s", absPath)
			break
		}
	}

	if migrationsPath == "" {
		return fmt.Errorf("migrations directory not found")
	}

	// Создаем экземпляр драйвера для Postgres
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Создаем экземпляр Migrate
	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Применяем миграции
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
