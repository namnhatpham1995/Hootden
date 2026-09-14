package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "",
		"API_ORIGIN":   "http://localhost:8080",
	})
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset")
	}
}

func TestLoad_RequiresAPIOrigin(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/hootden",
		"API_ORIGIN":   "",
	})
	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when API_ORIGIN is unset")
	}
	if got := err.Error(); got != "API_ORIGIN is required" {
		t.Errorf("error = %q, want it to name API_ORIGIN", got)
	}
}

// Table from design.md's "the startup check compares hosts, not sites".
func TestCheckCookieReachability(t *testing.T) {
	tests := []struct {
		name         string
		appOrigin    string
		apiOrigin    string
		cookieDomain string
		wantErr      bool
	}{
		{
			name:         "platform-issued hostnames on unrelated domains is refused",
			appOrigin:    "https://hootden.vercel.app",
			apiOrigin:    "https://hootden-prod.up.railway.app",
			cookieDomain: "",
			wantErr:      true,
		},
		{
			name:         "localhost dev with empty cookie domain is fine",
			appOrigin:    "http://localhost:3000",
			apiOrigin:    "http://localhost:8080",
			cookieDomain: "",
			wantErr:      false,
		},
		{
			name:         "apex and api subdomain covered by cookie domain is fine",
			appOrigin:    "https://hootden.example",
			apiOrigin:    "https://api.hootden.example",
			cookieDomain: "hootden.example",
			wantErr:      false,
		},
		{
			name:         "apex and api subdomain with no cookie domain is refused",
			appOrigin:    "https://hootden.example",
			apiOrigin:    "https://api.hootden.example",
			cookieDomain: "",
			wantErr:      true,
		},
		{
			name:         "api host outside the cookie domain is refused",
			appOrigin:    "https://hootden.example",
			apiOrigin:    "https://api.other.example",
			cookieDomain: "hootden.example",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCookieReachability(tc.appOrigin, tc.apiOrigin, tc.cookieDomain)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCheckCookieReachability_NamesConflictingValues(t *testing.T) {
	err := checkCookieReachability("https://hootden.vercel.app", "https://hootden-prod.up.railway.app", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()
	for _, want := range []string{"hootden.vercel.app", "hootden-prod.up.railway.app"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not name %q", msg, want)
		}
	}
}

func TestLoad_RefusesUnreachableCookieConfiguration(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL":  "postgres://localhost/hootden",
		"APP_ORIGIN":    "https://hootden.vercel.app",
		"API_ORIGIN":    "https://hootden-prod.up.railway.app",
		"COOKIE_DOMAIN": "",
	})
	if _, err := Load(); err == nil {
		t.Fatal("expected Load to refuse today's broken deployment shape")
	}
}

func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		if v == "" {
			os.Unsetenv(k)
			continue
		}
		os.Setenv(k, v)
	}
	t.Cleanup(func() {
		for k := range vars {
			os.Unsetenv(k)
		}
	})
}
