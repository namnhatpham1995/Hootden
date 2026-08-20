package page

import "testing"

func TestDelete_Leaf(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	n, err := Create(t.Context(), pool, denID, "", "Leaf")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delete(t.Context(), pool, n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("got %d nodes after delete, want 0", len(nodes))
	}
}

func TestDelete_SubtreeCascades(t *testing.T) {
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
	grandchild, err := Create(t.Context(), pool, denID, child.ID, "Grandchild")
	if err != nil {
		t.Fatalf("Create grandchild: %v", err)
	}

	if err := Delete(t.Context(), pool, parent.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	for _, n := range nodes {
		if n.ID == parent.ID || n.ID == child.ID || n.ID == grandchild.ID {
			t.Errorf("node %q still present after deleting its ancestor", n.ID)
		}
	}
	if len(nodes) != 0 {
		t.Errorf("got %d remaining nodes, want 0", len(nodes))
	}
}
