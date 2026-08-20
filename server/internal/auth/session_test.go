package auth

import (
	"errors"
	"testing"
	"time"
)

func createTestUser(t *testing.T) string {
	t.Helper()
	pool := testPool(t)
	sub, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	userID, _, err := GetOrCreateUserBySub(t.Context(), pool, sub, "session-test@example.com")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return userID
}

func TestSession_ValidTokenResolves(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)

	rawToken, err := CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if rawToken == "" {
		t.Fatal("CreateSession returned an empty token")
	}

	// The raw token must not be recoverable from what's stored.
	var stored string
	err = pool.QueryRow(t.Context(), `SELECT token_hash FROM sessions WHERE token_hash = $1`, hashToken(rawToken)).Scan(&stored)
	if err != nil {
		t.Fatalf("expected a stored row for the hash: %v", err)
	}
	if stored == rawToken {
		t.Error("stored token_hash equals the raw token; only the hash must be stored")
	}

	gotUserID, err := ResolveSession(t.Context(), pool, rawToken)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("ResolveSession user = %q, want %q", gotUserID, userID)
	}
}

func TestSession_UnknownTokenRejected(t *testing.T) {
	pool := testPool(t)

	bogus, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}

	_, err = ResolveSession(t.Context(), pool, bogus)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("ResolveSession err = %v, want ErrSessionNotFound", err)
	}
}

func TestSession_ExpiredTokenRejected(t *testing.T) {
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

	_, err = ResolveSession(t.Context(), pool, rawToken)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("ResolveSession err = %v, want ErrSessionNotFound for an expired session", err)
	}
}

func TestSession_RevokedTokenRejected(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)

	rawToken, err := CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := RevokeSession(t.Context(), pool, rawToken); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	_, err = ResolveSession(t.Context(), pool, rawToken)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("ResolveSession err = %v, want ErrSessionNotFound after sign-out", err)
	}
}
