package page

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrCycle means the requested move would make a page a descendant of
// itself.
var ErrCycle = errors.New("move would create a cycle")

// maxAncestorWalk bounds the cycle-guard walk. A real tree never gets
// close to this; it exists only so corrupted data can't hang the request.
const maxAncestorWalk = 1000

// evacuatedPosition is used only mid-transaction, for the instant between
// removing the moved page from its old slot and placing it in its new
// one. Never a value Create or a completed Move ever assigns, so it can't
// collide with a real row -- necessary because the moved page's own row
// would otherwise still occupy its old position while siblings are
// shifted into that same group.
const evacuatedPosition = -1

// Move repositions pageID to newPosition among the children of newParentID
// ("" for the top level), reindexing siblings so positions stay unique and
// gap-free within each group. The caller is expected to have already
// verified ownership of both pageID and newParentID (if non-empty) --
// see workspace.RequireOwnPage.
//
// Postgres checks a unique index per row, immediately, as each row is
// written -- not once at the end of a statement. A blind bulk
// "UPDATE ... SET position = position + 1 WHERE position >= N" can
// therefore collide mid-statement depending on the order Postgres happens
// to process rows in, even though the statement's final result would have
// been valid. Shifting is done as individually ordered single-row updates
// instead: descending by position for an upward shift (the highest slot,
// which is empty, is filled first), ascending for a downward shift (the
// lowest slot, freed by the page's departure, is filled first). Each
// individual update then always writes into a position nothing else
// currently holds.
func Move(ctx context.Context, pool *pgxpool.Pool, pageID, newParentID string, newPosition int) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var workspaceID string
		var oldParent sql.NullString
		var oldPosition int
		err := tx.QueryRow(ctx,
			`SELECT workspace_id, parent_id, position FROM pages WHERE id = $1 FOR UPDATE`,
			pageID,
		).Scan(&workspaceID, &oldParent, &oldPosition)
		if err != nil {
			return err
		}

		if newParentID != "" {
			cyclic, err := wouldCycle(ctx, tx, pageID, newParentID)
			if err != nil {
				return err
			}
			if cyclic {
				return ErrCycle
			}
		}

		if _, err := tx.Exec(ctx, `UPDATE pages SET position = $1 WHERE id = $2`, evacuatedPosition, pageID); err != nil {
			return err
		}

		// Close the gap the page left behind in its old group: shift
		// everything after it down by one, lowest position first.
		if err := shiftGroup(ctx, tx, workspaceID, oldParent.String, oldPosition, -1); err != nil {
			return err
		}

		// Make room at the target position in the (possibly same) new
		// group: shift everything from there onward up by one, highest
		// position first.
		if err := shiftGroup(ctx, tx, workspaceID, newParentID, newPosition, +1); err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`UPDATE pages SET parent_id = NULLIF($1, '')::uuid, position = $2 WHERE id = $3`,
			newParentID, newPosition, pageID,
		)
		return err
	})
}

// shiftGroup moves every page in (workspaceID, parentID) whose position is
// past threshold by delta (+1 or -1), one row at a time in the order that
// guarantees no individual update ever writes into a position another row
// in the set still occupies.
func shiftGroup(ctx context.Context, tx pgx.Tx, workspaceID, parentID string, threshold, delta int) error {
	var order string
	var cmp string
	if delta > 0 {
		// Shifting up: process highest-first so each row moves into a
		// slot the row above it just vacated (or, for the topmost row,
		// into the room made by evacuating the moved page).
		order = "DESC"
		cmp = ">="
	} else {
		// Shifting down: process lowest-first, same reasoning in reverse.
		order = "ASC"
		cmp = ">"
	}

	rows, err := tx.Query(ctx,
		`SELECT id FROM pages
		 WHERE workspace_id = $1 AND parent_id IS NOT DISTINCT FROM NULLIF($2, '')::uuid AND position `+cmp+` $3
		 ORDER BY position `+order,
		workspaceID, parentID, threshold,
	)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE pages SET position = position + $1 WHERE id = $2`, delta, id); err != nil {
			return err
		}
	}
	return nil
}

// wouldCycle reports whether newParentID is pageID itself or a descendant
// of it, by walking parent_id upward from newParentID.
func wouldCycle(ctx context.Context, tx pgx.Tx, pageID, newParentID string) (bool, error) {
	if newParentID == pageID {
		return true, nil
	}
	current := newParentID
	for i := 0; i < maxAncestorWalk; i++ {
		var parent sql.NullString
		err := tx.QueryRow(ctx, `SELECT parent_id FROM pages WHERE id = $1`, current).Scan(&parent)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !parent.Valid {
			return false, nil
		}
		if parent.String == pageID {
			return true, nil
		}
		current = parent.String
	}
	return true, errors.New("cycle guard exceeded max ancestor depth")
}
