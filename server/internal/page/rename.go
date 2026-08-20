package page

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Rename(ctx context.Context, pool *pgxpool.Pool, pageID, title string) error {
	if title == "" {
		title = untitledPlaceholder
	}
	_, err := pool.Exec(ctx, `UPDATE pages SET title = $1 WHERE id = $2`, title, pageID)
	return err
}
