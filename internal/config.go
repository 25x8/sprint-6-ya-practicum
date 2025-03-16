package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Accrual DB configuration
	AccrualDBHost     string
	AccrualDBPort     string
	AccrualDBUser     string
	AccrualDBPassword string
	AccrualDBName     string
	AccrualDBSSLMode  string

	// Gophermart DB configuration
	GophermartDBHost     string
	GophermartDBPort     string
	GophermartDBUser     string
	GophermartDBPassword string
	GophermartDBName     string
	GophermartDBSSLMode  string

	// System addresses
	AccrualSystemAddress string
}

func LoadConfig() (*Config, error) {

	err := godotenv.Load()

	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %v", err)
	}

	cfg := &Config{
		// Accrual DB configuration
		AccrualDBHost:     getEnv("ACCRUAL_DB_HOST", "localhost"),
		AccrualDBPort:     getEnv("ACCRUAL_DB_PORT", "5432"),
		AccrualDBUser:     getEnv("ACCRUAL_DB_USER", "accrual_user"),
		AccrualDBPassword: getEnv("ACCRUAL_DB_PASSWORD", "accrual_password"),
		AccrualDBName:     getEnv("ACCRUAL_DB_NAME", "accrual_db"),
		AccrualDBSSLMode:  getEnv("ACCRUAL_DB_SSL_MODE", "disable"),

		// Gophermart DB configuration
		GophermartDBHost:     getEnv("GOPHERMART_DB_HOST", "localhost"),
		GophermartDBPort:     getEnv("GOPHERMART_DB_PORT", "5432"),
		GophermartDBUser:     getEnv("GOPHERMART_DB_USER", "accrual_user"),
		GophermartDBPassword: getEnv("GOPHERMART_DB_PASSWORD", "accrual_password"),
		GophermartDBName:     getEnv("GOPHERMART_DB_NAME", "gophermart"),
		GophermartDBSSLMode:  getEnv("GOPHERMART_DB_SSL_MODE", "disable"),

		// System addresses
		AccrualSystemAddress: getEnv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8081"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
