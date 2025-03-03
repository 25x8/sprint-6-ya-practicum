package db

import (
	"fmt"
	"log"

	config "github.com/25x8/sprint-6-ya-practicum/internal"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

func ApplyMigrations() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.AccrualDBUser, cfg.AccrualDBPassword, cfg.AccrualDBHost, cfg.AccrualDBPort, cfg.AccrualDBName, cfg.AccrualDBSSLMode,
	)

	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)

	if err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	log.Println("Migration applied successfuly!")
	return nil
}
