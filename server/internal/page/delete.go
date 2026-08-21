package page

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Delete removes pageID. Descendants go with it via ON DELETE CASCADE.
func Delete(ctx context.Context, pool *pgxpool.Pool, pageID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM pages WHERE id = $1`, pageID)
	return err
}
