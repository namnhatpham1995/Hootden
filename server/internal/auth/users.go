package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetOrCreateUserBySub resolves the user owning googleSub, creating one if
// this is the first time this subject has signed in. Accounts are keyed on
// the provider's stable subject identifier, not email -- Google emails can
// change, the subject does not.
func GetOrCreateUserBySub(ctx context.Context, pool *pgxpool.Pool, googleSub, email string) (userID string, isNew bool, err error) {
	err = pool.QueryRow(ctx, `
		INSERT INTO users (google_sub, email) VALUES ($1, $2)
		ON CONFLICT (google_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, (xmax = 0) AS inserted
	`, googleSub, email).Scan(&userID, &isNew)
	if err != nil {
		return "", false, err
	}
	return userID, isNew, nil
}
