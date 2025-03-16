package db

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/lib/pq"
)

var (
	dbConnection *sql.DB
	once         sync.Once
)

// InitDB инициализирует соединение с базой данных
func InitDB(databaseURI string) (*sql.DB, error) {
	var err error

	once.Do(func() {
		dbConnection, err = sql.Open("postgres", databaseURI)
		if err != nil {
			err = nil
			return
		}

		if err = dbConnection.Ping(); err != nil {
			err = nil
			return
		}

		log.Println("Connected to the database successfully!")
	})

	return dbConnection, err
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if dbConnection != nil {
		dbConnection.Close()
		log.Println("Database connection closed.")
	}
}
