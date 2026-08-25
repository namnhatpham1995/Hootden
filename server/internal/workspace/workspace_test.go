package workspace

import "testing"

func TestEnsureUserAndDen_FirstSignIn(t *testing.T) {
	pool := testPool(t)
	sub := randomSub(t)
	email := randomEmail(t)

	userID, denID, isNew, err := EnsureUserAndDen(t.Context(), pool, sub, email)
	if err != nil {
		t.Fatalf("EnsureUserAndDen: %v", err)
	}
	if !isNew {
		t.Error("isNew = false, want true for a first sign-in")
	}
	if userID == "" || denID == "" {
		t.Fatalf("got empty ids: userID=%q denID=%q", userID, denID)
	}

	var personal bool
	var ownerID string
	err = pool.QueryRow(t.Context(),
		`SELECT owner_id, personal FROM workspaces WHERE id = $1`, denID,
	).Scan(&ownerID, &personal)
	if err != nil {
		t.Fatalf("query workspace: %v", err)
	}
	if ownerID != userID {
		t.Errorf("workspace owner_id = %q, want %q", ownerID, userID)
	}
	if !personal {
		t.Error("workspace personal = false, want true")
	}
}

func TestEnsureUserAndDen_ReturningSignInReusesDen(t *testing.T) {
	pool := testPool(t)
	sub := randomSub(t)
	email := randomEmail(t)

	userID1, denID1, isNew1, err := EnsureUserAndDen(t.Context(), pool, sub, email)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !isNew1 {
		t.Fatal("first call: isNew = false, want true")
	}

	userID2, denID2, isNew2, err := EnsureUserAndDen(t.Context(), pool, sub, email)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if isNew2 {
		t.Error("second call: isNew = true, want false")
	}
	if userID2 != userID1 {
		t.Errorf("second call user = %q, want %q", userID2, userID1)
	}
	if denID2 != denID1 {
		t.Errorf("second call den = %q, want %q (same Den, not a second workspace)", denID2, denID1)
	}

	var workspaceCount int
	err = pool.QueryRow(t.Context(),
		`SELECT count(*) FROM workspaces WHERE owner_id = $1`, userID1,
	).Scan(&workspaceCount)
	if err != nil {
		t.Fatalf("count workspaces: %v", err)
	}
	if workspaceCount != 1 {
		t.Errorf("workspace count for user = %d, want exactly 1", workspaceCount)
	}
}
