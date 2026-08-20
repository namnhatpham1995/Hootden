package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const appOrigin = "https://hootden.example"

func newCORSHandler() http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return CORS(appOrigin, inner)
}

func TestCORS_Preflight_MatchingOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/pages", nil)
	req.Header.Set("Origin", appOrigin)
	rec := httptest.NewRecorder()

	newCORSHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != appOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, appOrigin)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Errorf("Access-Control-Allow-Origin must never be \"*\" when credentials are allowed")
	}
}

func TestCORS_Preflight_MismatchedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/pages", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	newCORSHandler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty for a non-matching origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want empty for a non-matching origin", got)
	}
}

func TestCORS_ActualRequest_PassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", appOrigin)
	rec := httptest.NewRecorder()

	newCORSHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (request should reach the inner handler)", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != appOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, appOrigin)
	}
}
