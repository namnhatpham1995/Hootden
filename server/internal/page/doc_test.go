package page

import (
	"encoding/json"
	"reflect"
	"testing"
)

// docEqual compares two JSON documents by value, not by byte layout --
// Postgres re-serializes JSONB with its own canonical spacing.
func docEqual(t *testing.T, got json.RawMessage, want string) bool {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	return reflect.DeepEqual(g, w)
}

func TestGet_EmptyDocumentForNewPage(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)

	created, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	p, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(p.Doc) != "{}" {
		t.Errorf("doc = %s, want empty object", p.Doc)
	}
}

func TestSaveDoc_UpdatesTimestamp(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)
	created, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	before, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if err := SaveDoc(t.Context(), pool, created.ID, json.RawMessage(`{"type":"doc"}`)); err != nil {
		t.Fatalf("SaveDoc: %v", err)
	}

	after, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !docEqual(t, after.Doc, `{"type":"doc"}`) {
		t.Errorf("doc = %s, want the saved document", after.Doc)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updated_at = %v, want later than %v", after.UpdatedAt, before.UpdatedAt)
	}
}

func TestSaveDoc_LastWriteWinsNotMerged(t *testing.T) {
	pool := testPool(t)
	_, denID := testDen(t)
	created, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := SaveDoc(t.Context(), pool, created.ID, json.RawMessage(`{"type":"doc","content":["first"]}`)); err != nil {
		t.Fatalf("SaveDoc first: %v", err)
	}
	if err := SaveDoc(t.Context(), pool, created.ID, json.RawMessage(`{"type":"doc","content":["second"]}`)); err != nil {
		t.Fatalf("SaveDoc second: %v", err)
	}

	p, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !docEqual(t, p.Doc, `{"type":"doc","content":["second"]}`) {
		t.Errorf("doc = %s, want exactly the second write, not a merge", p.Doc)
	}
}
