package workspace

import (
	"errors"
	"testing"
)

func TestRequireOwnWorkspace(t *testing.T) {
	pool := testPool(t)
	ownerID, denID, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), "owner@example.com")
	if err != nil {
		t.Fatalf("setup owner: %v", err)
	}
	otherID, _, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), "other@example.com")
	if err != nil {
		t.Fatalf("setup other: %v", err)
	}

	if err := RequireOwnWorkspace(t.Context(), pool, ownerID, denID); err != nil {
		t.Errorf("owner: got %v, want nil", err)
	}
	if err := RequireOwnWorkspace(t.Context(), pool, otherID, denID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other account: got %v, want ErrNotFound", err)
	}
	if err := RequireOwnWorkspace(t.Context(), pool, ownerID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("nonexistent workspace: got %v, want ErrNotFound", err)
	}
}

func TestRequireOwnPage(t *testing.T) {
	pool := testPool(t)
	ownerID, denID, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), "owner@example.com")
	if err != nil {
		t.Fatalf("setup owner: %v", err)
	}
	otherID, _, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), "other@example.com")
	if err != nil {
		t.Fatalf("setup other: %v", err)
	}

	var pageID string
	err = pool.QueryRow(t.Context(),
		`INSERT INTO pages (workspace_id, title, position) VALUES ($1, 'Untitled', 0) RETURNING id`,
		denID,
	).Scan(&pageID)
	if err != nil {
		t.Fatalf("insert page: %v", err)
	}

	if err := RequireOwnPage(t.Context(), pool, ownerID, pageID); err != nil {
		t.Errorf("owner: got %v, want nil", err)
	}
	if err := RequireOwnPage(t.Context(), pool, otherID, pageID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other account's page: got %v, want ErrNotFound", err)
	}
	if err := RequireOwnPage(t.Context(), pool, ownerID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("nonexistent page: got %v, want ErrNotFound", err)
	}
}
