package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AccrualDBHost     string
	AccrualDBPort     string
	AccrualDBUser     string
	AccrualDBPassword string
	AccrualDBName     string
	AccrualDBSSLMode  string
}

func LoadConfig() (*Config, error) {

	err := godotenv.Load()

	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %v", err)
	}

	cfg := &Config{
		AccrualDBHost:     getEnv("ACCRUAL_DB_HOST", "localhost"),
		AccrualDBPort:     getEnv("ACCRUAL_DB_PORT", "5432"),
		AccrualDBUser:     getEnv("ACCRUAL_DB_USER", "accrual_user"),
		AccrualDBPassword: getEnv("ACCRUAL_DB_PASSWORD", "accrual_password"),
		AccrualDBName:     getEnv("ACCRUAL_DB_NAME", "accrual_db"),
		AccrualDBSSLMode:  getEnv("ACCRUAL_DB_SSL_MODE", "disable"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
