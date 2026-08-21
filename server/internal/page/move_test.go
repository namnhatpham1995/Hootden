package page

import (
	"errors"
	"testing"
)

func TestMove_ReorderAmongSiblings(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	a, err := Create(t.Context(), pool, denID, "", "A")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := Create(t.Context(), pool, denID, "", "B")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	c, err := Create(t.Context(), pool, denID, "", "C")
	if err != nil {
		t.Fatalf("create c: %v", err)
	}
	// Order: A(0) B(1) C(2). Move A to position 2 -> expect B(0) C(1) A(2).
	if err := Move(t.Context(), pool, a.ID, "", 2); err != nil {
		t.Fatalf("Move: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if byID[b.ID].Position != 0 {
		t.Errorf("B position = %d, want 0", byID[b.ID].Position)
	}
	if byID[c.ID].Position != 1 {
		t.Errorf("C position = %d, want 1", byID[c.ID].Position)
	}
	if byID[a.ID].Position != 2 {
		t.Errorf("A position = %d, want 2", byID[a.ID].Position)
	}
}

func TestMove_ReorderBackwards(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	a, err := Create(t.Context(), pool, denID, "", "A")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := Create(t.Context(), pool, denID, "", "B")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	c, err := Create(t.Context(), pool, denID, "", "C")
	if err != nil {
		t.Fatalf("create c: %v", err)
	}
	// Order: A(0) B(1) C(2). Move C to position 0 -> expect C(0) A(1) B(2).
	if err := Move(t.Context(), pool, c.ID, "", 0); err != nil {
		t.Fatalf("Move: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if byID[c.ID].Position != 0 {
		t.Errorf("C position = %d, want 0", byID[c.ID].Position)
	}
	if byID[a.ID].Position != 1 {
		t.Errorf("A position = %d, want 1", byID[a.ID].Position)
	}
	if byID[b.ID].Position != 2 {
		t.Errorf("B position = %d, want 2", byID[b.ID].Position)
	}
}

func TestMove_ReparentWithDescendants(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	oldParent, err := Create(t.Context(), pool, denID, "", "Old Parent")
	if err != nil {
		t.Fatalf("create oldParent: %v", err)
	}
	newParent, err := Create(t.Context(), pool, denID, "", "New Parent")
	if err != nil {
		t.Fatalf("create newParent: %v", err)
	}
	moved, err := Create(t.Context(), pool, denID, oldParent.ID, "Moved")
	if err != nil {
		t.Fatalf("create moved: %v", err)
	}
	grandchild, err := Create(t.Context(), pool, denID, moved.ID, "Grandchild")
	if err != nil {
		t.Fatalf("create grandchild: %v", err)
	}

	if err := Move(t.Context(), pool, moved.ID, newParent.ID, 0); err != nil {
		t.Fatalf("Move: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if byID[moved.ID].ParentID != newParent.ID {
		t.Errorf("moved parent_id = %q, want %q", byID[moved.ID].ParentID, newParent.ID)
	}
	// The subtree moved with it -- grandchild's parent is unchanged.
	if byID[grandchild.ID].ParentID != moved.ID {
		t.Errorf("grandchild parent_id = %q, want %q (unchanged)", byID[grandchild.ID].ParentID, moved.ID)
	}
}

func TestMove_OldSiblingGapCloses(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	newParent, err := Create(t.Context(), pool, denID, "", "New Parent")
	if err != nil {
		t.Fatalf("create newParent: %v", err)
	}
	a, err := Create(t.Context(), pool, denID, "", "A")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := Create(t.Context(), pool, denID, "", "B")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	// Root order: newParent(0) A(1) B(2). Move A under newParent.
	if err := Move(t.Context(), pool, a.ID, newParent.ID, 0); err != nil {
		t.Fatalf("Move: %v", err)
	}

	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	// B should have shifted down to close the gap left by A -- newParent
	// still holds root position 0, so B moves from 2 to 1.
	if byID[b.ID].Position != 1 {
		t.Errorf("B position after A moved away = %d, want 1", byID[b.ID].Position)
	}
}

func TestMove_CyclicMoveRefused(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	parent, err := Create(t.Context(), pool, denID, "", "Parent")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := Create(t.Context(), pool, denID, parent.ID, "Child")
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	err = Move(t.Context(), pool, parent.ID, child.ID, 0)
	if !errors.Is(err, ErrCycle) {
		t.Fatalf("Move(parent under its own child) = %v, want ErrCycle", err)
	}

	// Tree unchanged.
	nodes, err := Tree(t.Context(), pool, denID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if byID[parent.ID].ParentID != "" {
		t.Errorf("parent parent_id = %q after refused move, want unchanged (empty)", byID[parent.ID].ParentID)
	}
	if byID[child.ID].ParentID != parent.ID {
		t.Errorf("child parent_id = %q after refused move, want unchanged (%q)", byID[child.ID].ParentID, parent.ID)
	}
}

func TestMove_SelfParentRefused(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	n, err := Create(t.Context(), pool, denID, "", "Self")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := Move(t.Context(), pool, n.ID, n.ID, 0); !errors.Is(err, ErrCycle) {
		t.Fatalf("Move(self under self) = %v, want ErrCycle", err)
	}
}
