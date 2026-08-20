package auth

import "testing"

func TestGetOrCreateUserBySub(t *testing.T) {
	pool := testPool(t)
	ctx := t.Context()

	sub, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}

	id1, isNew1, err := GetOrCreateUserBySub(ctx, pool, sub, "a@example.com")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !isNew1 {
		t.Error("first call: isNew = false, want true")
	}
	if id1 == "" {
		t.Error("first call: got empty user id")
	}

	id2, isNew2, err := GetOrCreateUserBySub(ctx, pool, sub, "a@example.com")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if isNew2 {
		t.Error("second call: isNew = true, want false (repeat subject should reuse the account)")
	}
	if id2 != id1 {
		t.Errorf("second call: user id = %q, want %q (same account)", id2, id1)
	}
}
