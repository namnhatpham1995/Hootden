package page

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
)

// currentParentID returns pageID's current parent ("" for the top level),
// used when a move only changes position and parent_id is left unspecified.
func currentParentID(ctx context.Context, pool *pgxpool.Pool, pageID string) (string, error) {
	var parent sql.NullString
	err := pool.QueryRow(ctx, `SELECT parent_id FROM pages WHERE id = $1`, pageID).Scan(&parent)
	if err != nil {
		return "", err
	}
	return parent.String, nil
}
