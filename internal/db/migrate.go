package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

// ApplyMigrations применяет миграции к базе данных
func ApplyMigrations(databaseURI string) error {
	// Определяем тип сервиса по имени исполняемого файла
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeName := filepath.Base(exe)
	log.Printf("Executable name: %s", exeName)

	// Определяем тип миграций на основе имени исполняемого файла
	var migrationType string
	if strings.Contains(exeName, "accrual") {
		migrationType = "accrual"
	} else {
		migrationType = "gophermart"
	}
	log.Printf("Using migration type: %s", migrationType)

	// Пытаемся обновить ограничение CHECK для поля status в таблице orders,
	// если это сервис accrual
	if migrationType == "accrual" {
		db, err := sql.Open("postgres", databaseURI)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		_, err = db.Exec(`
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
	}

	// Ищем директорию с миграциями
	migrationsDirs := []string{
		fmt.Sprintf("migrations/%s", migrationType),
		fmt.Sprintf("../migrations/%s", migrationType),
		fmt.Sprintf("../../migrations/%s", migrationType),
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

	log.Printf("Migration path: %s", migrationsPath)
	log.Printf("Database URI: %s", databaseURI)

	m, err := migrate.New(migrationsPath, databaseURI)
	if err != nil {
		// Альтернативный подход с использованием WithInstance
		db, err := sql.Open("postgres", databaseURI)
		if err != nil {
			return fmt.Errorf("failed to connect to database for migration: %w", err)
		}
		defer db.Close()

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("failed to create postgres driver: %w", err)
		}

		m, err = migrate.NewWithDatabaseInstance(
			migrationsPath,
			"postgres", driver)
		if err != nil {
			return fmt.Errorf("failed to create migration instance: %w", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Migration applied successfully!")
	return nil
}
