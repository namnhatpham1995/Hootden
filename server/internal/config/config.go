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
	// GoogleClientID and GoogleClientSecret identify this app to Google's
	// OAuth 2.0 endpoints.
	GoogleClientID     string
	GoogleClientSecret string
	// GoogleRedirectURL is this API's own OAuth callback URL, registered as
	// an authorized redirect URI on the Google OAuth client.
	GoogleRedirectURL string
	// CookieDomain is the Domain attribute for the session and OAuth-flow
	// cookies, e.g. ".hootden.example" so it's shared by the apex and the
	// api. subdomain. Empty means a host-only cookie, which is what local
	// development against localhost needs.
	CookieDomain string
}

func Load() (Config, error) {
	cfg := Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		AppOrigin:          getEnv("APP_ORIGIN", "http://localhost:3000"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		CookieDomain:       os.Getenv("COOKIE_DOMAIN"),
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
