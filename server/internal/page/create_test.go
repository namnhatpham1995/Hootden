package page

import "testing"

func TestCreate_RootPage(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	n, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.Title != "My Page" {
		t.Errorf("title = %q, want %q", n.Title, "My Page")
	}
	if n.ParentID != "" {
		t.Errorf("parent_id = %q, want empty (root)", n.ParentID)
	}
	if n.Position != 0 {
		t.Errorf("position = %d, want 0 for the first page", n.Position)
	}
}

func TestCreate_NestedPage(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	parent, err := Create(t.Context(), pool, denID, "", "Parent")
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	child, err := Create(t.Context(), pool, denID, parent.ID, "Child")
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Errorf("child parent_id = %q, want %q", child.ParentID, parent.ID)
	}
	if child.Position != 0 {
		t.Errorf("child position = %d, want 0 (first child)", child.Position)
	}
}

func TestCreate_UntitledPlaceholder(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	n, err := Create(t.Context(), pool, denID, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.Title != untitledPlaceholder {
		t.Errorf("title = %q, want placeholder %q", n.Title, untitledPlaceholder)
	}
}

func TestCreate_SiblingsGetIncrementingPositions(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	first, err := Create(t.Context(), pool, denID, "", "First")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := Create(t.Context(), pool, denID, "", "Second")
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if first.Position != 0 || second.Position != 1 {
		t.Errorf("positions = %d, %d, want 0, 1", first.Position, second.Position)
	}
}
