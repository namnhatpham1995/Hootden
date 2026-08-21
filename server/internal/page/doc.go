package page

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Page is a single page including its document body -- returned only by
// the single-page fetch, never by the tree listing.
type Page struct {
	ID        string          `json:"id"`
	ParentID  string          `json:"parent_id,omitempty"`
	Title     string          `json:"title"`
	Doc       json.RawMessage `json:"doc"`
	Position  int             `json:"position"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Get fetches a single page including its document body.
func Get(ctx context.Context, pool *pgxpool.Pool, pageID string) (Page, error) {
	var p Page
	var parentID sql.NullString
	err := pool.QueryRow(ctx, `
		SELECT id, parent_id, title, doc, position, updated_at
		FROM pages WHERE id = $1
	`, pageID).Scan(&p.ID, &parentID, &p.Title, &p.Doc, &p.Position, &p.UpdatedAt)
	if err != nil {
		return Page{}, err
	}
	p.ParentID = parentID.String
	return p, nil
}

// SaveDoc overwrites a page's document body and bumps updated_at. The last
// call to reach the database wins -- there is no merge, matching the
// single-account concurrent-edit requirement.
func SaveDoc(ctx context.Context, pool *pgxpool.Pool, pageID string, doc json.RawMessage) error {
	_, err := pool.Exec(ctx, `UPDATE pages SET doc = $1, updated_at = now() WHERE id = $2`, doc, pageID)
	return err
}
