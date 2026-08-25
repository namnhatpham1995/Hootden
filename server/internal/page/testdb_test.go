package page

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/migrate"
	"github.com/namnhatpham1995/Hootden/server/internal/workspace"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	if err := migrate.Up(databaseURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func randomSub(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// testDen creates a fresh user + Den, returning (userID, denID).
func testDen(t *testing.T) (string, string) {
	t.Helper()
	pool := testPool(t)
	userID, denID, _, err := workspace.EnsureUserAndDen(t.Context(), pool, randomSub(t), randomSub(t)+"@example.com")
	if err != nil {
		t.Fatalf("testDen setup: %v", err)
	}
	return userID, denID
}
