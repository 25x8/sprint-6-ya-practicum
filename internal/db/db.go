package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	config "github.com/25x8/sprint-6-ya-practicum/internal"
)

var (
	dbConnection *sql.DB
	once sync.Once
)

func InitDB() (*sql.DB, error) {

	var err error

	once.Do(func() {
		cfg, cfgErr := config.LoadConfig()
		if cfgErr != nil {
			err = fmt.Errorf("failed to load config: &v")
			return
		}

		dbURL := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.AccrualDBUser, cfg.AccrualDBPassword, cfg.AccrualDBHost, cfg.AccrualDBPort, cfg.AccrualDBName, cfg.AccrualDBSSLMode,
		)

		dbConnection, err = sql.Open("postgres", dbURL)
		if err != nil {
			err = fmt.Errorf("failed to connect to database: %v", err)
			return
		}

		if err = dbConnection.Ping(); err != nil {
			err = fmt.Errorf("database is unreachable: %v", err)
			return
		}

		log.Println("Connected to the database successfully!")
		
	})

	return dbConnection, err
}

func CloseDB() {
	if dbConnection != nil {
		dbConnection.Close()
		log.Println("Database connection closed.")
	}
}