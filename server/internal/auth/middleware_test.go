package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testCookieDomain = ""

func withSessionCookie(req *http.Request, rawToken string) *http.Request {
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawToken})
	return req
}

func TestRequireAuth_ValidSession(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)
	rawToken, err := CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var gotUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := withSessionCookie(httptest.NewRequest(http.MethodGet, "/me", nil), rawToken)
	rec := httptest.NewRecorder()
	RequireAuth(pool, testCookieDomain)(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUserID != userID {
		t.Errorf("context user id = %q, want %q", gotUserID, userID)
	}
}

func TestRequireAuth_NoCookie(t *testing.T) {
	pool := testPool(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not run without a session")
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	RequireAuth(pool, testCookieDomain)(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_UnknownToken(t *testing.T) {
	pool := testPool(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not run for an unknown token")
	})

	bogus, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}

	req := withSessionCookie(httptest.NewRequest(http.MethodGet, "/me", nil), bogus)
	rec := httptest.NewRecorder()
	RequireAuth(pool, testCookieDomain)(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Result().Cookies(); len(got) == 0 || got[0].MaxAge >= 0 {
		t.Error("expected the stale session cookie to be cleared (MaxAge < 0)")
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)

	rawToken, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	_, err = pool.Exec(t.Context(),
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(rawToken), userID, time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not run for an expired token")
	})

	req := withSessionCookie(httptest.NewRequest(http.MethodGet, "/me", nil), rawToken)
	rec := httptest.NewRecorder()
	RequireAuth(pool, testCookieDomain)(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
