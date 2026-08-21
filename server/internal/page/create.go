package page

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const untitledPlaceholder = "Untitled"

// Create adds a page as the last child of parentID (or at the top level if
// parentID is empty), defaulting its title if none is given.
func Create(ctx context.Context, pool *pgxpool.Pool, workspaceID, parentID, title string) (Node, error) {
	if title == "" {
		title = untitledPlaceholder
	}

	var n Node
	err := pool.QueryRow(ctx, `
		INSERT INTO pages (workspace_id, parent_id, title, position)
		SELECT $1, NULLIF($2, '')::uuid, $3,
			COALESCE(MAX(position) + 1, 0)
		FROM pages
		WHERE workspace_id = $1 AND parent_id IS NOT DISTINCT FROM NULLIF($2, '')::uuid
		RETURNING id, title, position
	`, workspaceID, parentID, title).Scan(&n.ID, &n.Title, &n.Position)
	if err != nil {
		return Node{}, err
	}
	n.ParentID = parentID
	return n, nil
}
