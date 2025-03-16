package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

// ApplyMigrations применяет миграции к базе данных
func ApplyMigrations(databaseURI string) error {
	// Определяем, какой сервис запускается (accrual или gophermart)
	// по названию исполняемого файла
	executable, err := os.Executable()
	if err != nil {
		log.Printf("Warning: Cannot determine executable name: %v, using default migrations", err)
		executable = ""
	}

	execName := filepath.Base(executable)
	log.Printf("Executable name: %s", execName)

	migrationType := "gophermart" // по умолчанию
	if strings.Contains(strings.ToLower(execName), "accrual") {
		migrationType = "accrual"
	}

	log.Printf("Using migration type: %s", migrationType)

	// Ищем миграции в нескольких возможных местах
	possiblePaths := []string{
		"./migrations/" + migrationType,
		"../migrations/" + migrationType,
		"../../migrations/" + migrationType,
		"./cmd/" + migrationType + "/migrations",
	}

	var migrationsPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			absPath, _ := filepath.Abs(path)
			migrationsPath = "file://" + absPath
			log.Printf("Using migrations from: %s", absPath)
			break
		}
	}

	if migrationsPath == "" {
		log.Println("Warning: Specialized migrations directory not found, trying default migrations")
		// Пробуем основную директорию миграций
		defaultPaths := []string{
			"./migrations",
			"../migrations",
			"../../migrations",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				absPath, _ := filepath.Abs(path)
				migrationsPath = "file://" + absPath
				log.Printf("Using default migrations from: %s", absPath)
				break
			}
		}
	}

	if migrationsPath == "" {
		log.Println("Warning: Migrations directory not found, using fallback path")
		migrationsPath = "file://migrations"
	}

	log.Printf("Migration path: %s", migrationsPath)
	log.Printf("Database URI: %s", databaseURI)

	m, err := migrate.New(
		migrationsPath,
		databaseURI,
	)

	if err != nil {
		return fmt.Errorf("migration initialization failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	log.Println("Migration applied successfully!")
	return nil
}
