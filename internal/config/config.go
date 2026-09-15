package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	DBMaxOpenConns int
	DBMaxIdleConns int
}

func Load() Config {
	return Config{
		Port:           envOrDefault("PORT", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
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
