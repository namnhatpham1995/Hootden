package workspace

import (
	"errors"
	"testing"
)

func TestCreateUserWithPassword_FreshEmailSucceeds(t *testing.T) {
	pool := testPool(t)
	email := randomEmail(t)

	userID, denID, err := CreateUserWithPassword(t.Context(), pool, email, "hashed")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
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

func TestCreateUserWithPassword_DuplicateEmailRejected(t *testing.T) {
	pool := testPool(t)
	email := randomEmail(t)

	if _, _, err := CreateUserWithPassword(t.Context(), pool, email, "hashed"); err != nil {
		t.Fatalf("first CreateUserWithPassword: %v", err)
	}

	if _, _, err := CreateUserWithPassword(t.Context(), pool, email, "other-hash"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate email: got %v, want ErrEmailTaken", err)
	}
}

func TestFindUserByEmail(t *testing.T) {
	pool := testPool(t)
	email := randomEmail(t)
	wantUserID, _, err := CreateUserWithPassword(t.Context(), pool, email, "hashed")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	userID, passwordHash, err := FindUserByEmail(t.Context(), pool, email)
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if userID != wantUserID {
		t.Errorf("userID = %q, want %q", userID, wantUserID)
	}
	if passwordHash != "hashed" {
		t.Errorf("passwordHash = %q, want %q", passwordHash, "hashed")
	}

	if _, _, err := FindUserByEmail(t.Context(), pool, randomEmail(t)); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("lookup miss: got %v, want ErrAccountNotFound", err)
	}
}

func TestFindUserByEmail_GoogleOnlyAccountHasEmptyHash(t *testing.T) {
	pool := testPool(t)
	email := randomEmail(t)
	if _, _, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), email); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, passwordHash, err := FindUserByEmail(t.Context(), pool, email)
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if passwordHash != "" {
		t.Errorf("passwordHash = %q, want empty for a Google-only account", passwordHash)
	}
}
