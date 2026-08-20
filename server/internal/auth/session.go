package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionTTL is the maximum lifetime of an issued session. The accounts
// spec requires expiry no later than 30 days.
const SessionTTL = 30 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found")

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// CreateSession issues a new session for userID and stores only its hash --
// a database leak does not hand out live sessions.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID string) (rawToken string, err error) {
	rawToken, err = randomToken()
	if err != nil {
		return "", err
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(rawToken), userID, time.Now().Add(SessionTTL),
	)
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

// ResolveSession returns the user ID owning rawToken, if it names an
// unexpired session. It never trusts the token's contents alone.
func ResolveSession(ctx context.Context, pool *pgxpool.Pool, rawToken string) (userID string, err error) {
	err = pool.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		hashToken(rawToken),
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

// RevokeSession deletes the session server-side, so any later request
// presenting the same token is rejected regardless of the cookie.
func RevokeSession(ctx context.Context, pool *pgxpool.Pool, rawToken string) error {
	_, err := pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(rawToken))
	return err
}
