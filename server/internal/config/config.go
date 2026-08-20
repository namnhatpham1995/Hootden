// Package config loads server configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	// Port the HTTP server listens on.
	Port string
	// DatabaseURL is a Postgres connection string, e.g.
	// postgres://user:pass@host:5432/dbname.
	DatabaseURL string
	// AppOrigin is the exact origin (scheme + host) of the frontend allowed
	// to make credentialed cross-origin requests to this API.
	AppOrigin string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AppOrigin:   getEnv("APP_ORIGIN", "http://localhost:3000"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
