package page

import "testing"

func TestTree_ReflectsStructureAndOrder(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	root1, err := Create(t.Context(), pool, denID, "", "Root 1")
	if err != nil {
		t.Fatalf("create root1: %v", err)
	}
	root2, err := Create(t.Context(), pool, denID, "", "Root 2")
	if err != nil {
		t.Fatalf("create root2: %v", err)
	}
	child, err := Create(t.Context(), pool, denID, root1.ID, "Child")
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}

	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if byID[root1.ID].ParentID != "" {
		t.Errorf("root1 parent_id = %q, want empty", byID[root1.ID].ParentID)
	}
	if byID[root2.ID].ParentID != "" {
		t.Errorf("root2 parent_id = %q, want empty", byID[root2.ID].ParentID)
	}
	if byID[child.ID].ParentID != root1.ID {
		t.Errorf("child parent_id = %q, want %q", byID[child.ID].ParentID, root1.ID)
	}
}

func TestTree_EmptyWorkspace(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0 for a fresh Den", len(nodes))
	}
}
