package workspace

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound means either the resource doesn't exist or userID doesn't own
// it. The two are indistinguishable on purpose: existence of another
// account's workspace or page must never be disclosed.
var ErrNotFound = errors.New("not found")

// RequireOwnWorkspace fails with ErrNotFound unless workspaceID exists and
// is owned by userID.
func RequireOwnWorkspace(ctx context.Context, pool *pgxpool.Pool, userID, workspaceID string) error {
	var owned bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1 AND owner_id = $2)`,
		workspaceID, userID,
	).Scan(&owned)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return nil
}

// RequireOwnPage fails with ErrNotFound unless pageID exists and belongs to
// a workspace owned by userID. Every page handler is expected to call this
// (or rely on a query already scoped the same way) before acting.
func RequireOwnPage(ctx context.Context, pool *pgxpool.Pool, userID, pageID string) error {
	var owned bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pages p
			JOIN workspaces w ON w.id = p.workspace_id
			WHERE p.id = $1 AND w.owner_id = $2
		)
	`, pageID, userID).Scan(&owned)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return nil
}
