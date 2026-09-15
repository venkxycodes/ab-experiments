package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	StorageBackend string
	DatabaseURL    string
	DBMaxOpenConns int
	DBMaxIdleConns int
}

func Load() Config {
	databaseURL := os.Getenv("DATABASE_URL")
	backend := os.Getenv("STORAGE_BACKEND")
	if backend == "" {
		backend = "memory"
		if databaseURL != "" {
			backend = "postgres"
		}
	}

	return Config{
		Port:           envOrDefault("PORT", ":8080"),
		StorageBackend: backend,
		DatabaseURL:    databaseURL,
		DBMaxOpenConns: envIntOrDefault("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns: envIntOrDefault("DB_MAX_IDLE_CONNS", 5),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
