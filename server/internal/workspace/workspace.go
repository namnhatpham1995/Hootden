// Package workspace manages Dens -- the personal workspace automatically
// provisioned per account -- and page ownership checks.
package workspace

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserResolver adapts EnsureUserAndDen to auth.UserResolver, so the
// callback handler can create a user's Den without the auth package
// depending on the workspace package.
type UserResolver struct {
	Pool *pgxpool.Pool
}

func (r UserResolver) ResolveUser(ctx context.Context, googleSub, email string) (userID string, err error) {
	userID, _, _, err = EnsureUserAndDen(ctx, r.Pool, googleSub, email)
	return userID, err
}

// EnsureUserAndDen resolves the user owning googleSub, creating both the
// user and their personal Den in one transaction if this is their first
// sign-in. A returning user's existing Den is looked up instead -- every
// account has exactly one, and it's never created twice.
func EnsureUserAndDen(ctx context.Context, pool *pgxpool.Pool, googleSub, email string) (userID, denID string, isNewUser bool, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (google_sub, email) VALUES ($1, $2)
			ON CONFLICT (google_sub) DO UPDATE SET email = EXCLUDED.email
			RETURNING id, (xmax = 0) AS inserted
		`, googleSub, email).Scan(&userID, &isNewUser); err != nil {
			return err
		}

		if isNewUser {
			return tx.QueryRow(ctx,
				`INSERT INTO workspaces (owner_id, personal) VALUES ($1, true) RETURNING id`,
				userID,
			).Scan(&denID)
		}
		return tx.QueryRow(ctx,
			`SELECT id FROM workspaces WHERE owner_id = $1 AND personal = true`,
			userID,
		).Scan(&denID)
	})
	if err != nil {
		return "", "", false, err
	}
	return userID, denID, isNewUser, nil
}
