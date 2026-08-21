package page

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Tree returns every page in workspaceID, without document bodies. The
// caller assembles parent/child structure from parent_id -- this is
// deliberately a flat list, not a nested tree, matching how little data
// this actually is at Den scale.
func Tree(ctx context.Context, pool *pgxpool.Pool, workspaceID string) ([]Node, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, parent_id, title, position FROM pages WHERE workspace_id = $1 ORDER BY position`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		if err := rows.Scan(&n.ID, &parentID, &n.Title, &n.Position); err != nil {
			return nil, err
		}
		n.ParentID = parentID.String
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
