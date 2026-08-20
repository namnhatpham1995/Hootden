package page

import "testing"

func TestRename(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	n, err := Create(t.Context(), pool, denID, "", "Old Title")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Rename(t.Context(), pool, n.ID, "New Title"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if nodes[0].Title != "New Title" {
		t.Errorf("title = %q, want %q", nodes[0].Title, "New Title")
	}
}
