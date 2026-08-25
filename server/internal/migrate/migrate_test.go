package migrate

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	if err := Up(databaseURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// randomEmail returns a unique email so repeated runs against a persisted
// dev database don't collide on the users.email UNIQUE constraint.
func randomEmail(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b) + "@example.com"
}

// TestPasswordAuthSchema exercises the 0002 migration's constraints
// directly, ahead of any application code that will rely on them.
func TestPasswordAuthSchema(t *testing.T) {
	pool := testPool(t)

	t.Run("password-only account inserts cleanly", func(t *testing.T) {
		var id string
		err := pool.QueryRow(t.Context(),
			`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
			randomEmail(t), "bcrypt-hash-placeholder",
		).Scan(&id)
		if err != nil {
			t.Fatalf("insert with password_hash only: %v", err)
		}
	})

	t.Run("account with neither auth method is rejected", func(t *testing.T) {
		_, err := pool.Exec(t.Context(),
			`INSERT INTO users (email) VALUES ($1)`,
			randomEmail(t),
		)
		if err == nil {
			t.Fatal("expected the users_has_auth_method check constraint to reject this row")
		}
	})

	t.Run("duplicate email is rejected", func(t *testing.T) {
		email := randomEmail(t)
		_, err := pool.Exec(t.Context(),
			`INSERT INTO users (email, password_hash) VALUES ($1, $2)`,
			email, "hash-one",
		)
		if err != nil {
			t.Fatalf("first insert: %v", err)
		}
		_, err = pool.Exec(t.Context(),
			`INSERT INTO users (email, password_hash) VALUES ($1, $2)`,
			email, "hash-two",
		)
		if err == nil {
			t.Fatal("expected the users_email_key unique constraint to reject the duplicate")
		}
	})
}
