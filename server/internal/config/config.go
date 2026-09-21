// Package config loads server configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
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
	// APIOrigin is this API's own public origin, as a browser reaching it
	// from AppOrigin sees it. Used only to check that the session cookie can
	// actually reach the app -- see the cookie-reachability check in Load.
	APIOrigin string
	// GoogleClientID and GoogleClientSecret identify this app to Google's
	// OAuth 2.0 endpoints.
	GoogleClientID     string
	GoogleClientSecret string
	// GoogleRedirectURL is this API's own OAuth callback URL, registered as
	// an authorized redirect URI on the Google OAuth client.
	GoogleRedirectURL string
	// CookieDomain is the Domain attribute for the session and OAuth-flow
	// cookies, e.g. "hootden.example" (no leading dot -- checkCookieReachability
	// matches it exactly, and a leading dot would make even the apex host fail
	// that check) so it's shared by the apex and the api. subdomain. Empty
	// means a host-only cookie, which is what local development against
	// localhost needs.
	CookieDomain string
}

func Load() (Config, error) {
	cfg := Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		AppOrigin:          getEnv("APP_ORIGIN", "http://localhost:3000"),
		APIOrigin:          os.Getenv("API_ORIGIN"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		CookieDomain:       os.Getenv("COOKIE_DOMAIN"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.APIOrigin == "" {
		return Config{}, fmt.Errorf("API_ORIGIN is required")
	}
	if err := checkCookieReachability(cfg.AppOrigin, cfg.APIOrigin, cfg.CookieDomain); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// checkCookieReachability refuses a configuration in which the session
// cookie the API issues could never be sent back to it from the app, so the
// failure surfaces at startup rather than as a browser silently discarding
// the cookie on every sign-in. See design.md's "the startup check compares
// hosts, not sites": with COOKIE_DOMAIN empty the cookie is host-only, so
// the app and API hosts must match exactly; with COOKIE_DOMAIN set, both
// hosts must equal it or be a subdomain of it.
func checkCookieReachability(appOrigin, apiOrigin, cookieDomain string) error {
	appHost, err := hostOf(appOrigin)
	if err != nil {
		return fmt.Errorf("APP_ORIGIN=%q is invalid: %w", appOrigin, err)
	}
	apiHost, err := hostOf(apiOrigin)
	if err != nil {
		return fmt.Errorf("API_ORIGIN=%q is invalid: %w", apiOrigin, err)
	}

	var reachable bool
	if cookieDomain == "" {
		reachable = appHost == apiHost
	} else {
		reachable = coveredByCookieDomain(appHost, cookieDomain) && coveredByCookieDomain(apiHost, cookieDomain)
	}
	if !reachable {
		return fmt.Errorf(
			"session cookie cannot reach the app: a browser would not send it from APP_ORIGIN=%s to API_ORIGIN=%s with COOKIE_DOMAIN=%q",
			appOrigin, apiOrigin, cookieDomain,
		)
	}
	return nil
}

func hostOf(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("no host in %q", origin)
	}
	return u.Hostname(), nil
}

func coveredByCookieDomain(host, cookieDomain string) bool {
	return host == cookieDomain || strings.HasSuffix(host, "."+cookieDomain)
}
