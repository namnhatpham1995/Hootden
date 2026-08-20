package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytes_RejectsOversizedBody(t *testing.T) {
	var readErr error
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	body := strings.NewReader(strings.Repeat("a", MaxBodyBytes+1))
	req := httptest.NewRequest(http.MethodPost, "/pages/123", body)
	rec := httptest.NewRecorder()

	MaxBytes(inner).ServeHTTP(rec, req)

	if readErr == nil {
		t.Fatal("expected reading an oversized body to error, got nil")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestMaxBytes_AllowsBodyAtLimit(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Errorf("unexpected read error for a body at the limit: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	})

	body := strings.NewReader(strings.Repeat("a", MaxBodyBytes))
	req := httptest.NewRequest(http.MethodPost, "/pages/123", body)
	rec := httptest.NewRecorder()

	MaxBytes(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
