package page

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
)

func sessionRequest(t *testing.T, pool *pgxpool.Pool, userID, method, target string, body any) *http.Request {
	t.Helper()
	rawToken, err := auth.CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, target, reqBody)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: rawToken})
	return req
}

func newMux(pool *pgxpool.Pool) http.Handler {
	requireAuth := auth.RequireAuth(pool, "")
	h := Handlers{Pool: pool}

	mux := http.NewServeMux()
	mux.Handle("GET /pages", requireAuth(http.HandlerFunc(h.List)))
	mux.Handle("POST /pages", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /pages/{id}", requireAuth(http.HandlerFunc(h.Get)))
	mux.Handle("PATCH /pages/{id}", requireAuth(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /pages/{id}", requireAuth(http.HandlerFunc(h.Delete)))
	return mux
}

func TestHandlers_Create_ParentOwnedByAnotherAccount_404(t *testing.T) {
	pool := testPool(t)
	_, otherDenID := testDen(t)
	otherPage, err := Create(t.Context(), pool, otherDenID, "", "Someone else's page")
	if err != nil {
		t.Fatalf("create other page: %v", err)
	}

	callerID, _ := testDen(t)
	req := sessionRequest(t, pool, callerID, http.MethodPost, "/pages", createRequest{ParentID: otherPage.ID, Title: "Mine"})
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (not 403 -- existence must not be disclosed)", rec.Code, http.StatusNotFound)
	}
}

func TestHandlers_Update_AnotherAccountsPage_404(t *testing.T) {
	pool := testPool(t)
	_, otherDenID := testDen(t)
	otherPage, err := Create(t.Context(), pool, otherDenID, "", "Someone else's page")
	if err != nil {
		t.Fatalf("create other page: %v", err)
	}

	callerID, _ := testDen(t)
	newTitle := "Hijacked"
	req := sessionRequest(t, pool, callerID, http.MethodPatch, "/pages/"+otherPage.ID, updateRequest{Title: &newTitle})
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Confirm nothing changed.
	nodes, err := Tree(t.Context(), pool, otherDenID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if nodes[0].Title != "Someone else's page" {
		t.Errorf("title = %q, want unchanged", nodes[0].Title)
	}
}

func TestHandlers_Delete_AnotherAccountsPage_404(t *testing.T) {
	pool := testPool(t)
	_, otherDenID := testDen(t)
	otherPage, err := Create(t.Context(), pool, otherDenID, "", "Someone else's page")
	if err != nil {
		t.Fatalf("create other page: %v", err)
	}

	callerID, _ := testDen(t)
	req := sessionRequest(t, pool, callerID, http.MethodDelete, "/pages/"+otherPage.ID, nil)
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	nodes, err := Tree(t.Context(), pool, otherDenID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("got %d nodes, want the page to still exist", len(nodes))
	}
}

func TestHandlers_Get_AnotherAccountsPage_404(t *testing.T) {
	pool := testPool(t)
	_, otherDenID := testDen(t)
	otherPage, err := Create(t.Context(), pool, otherDenID, "", "Someone else's page")
	if err != nil {
		t.Fatalf("create other page: %v", err)
	}

	callerID, _ := testDen(t)
	req := sessionRequest(t, pool, callerID, http.MethodGet, "/pages/"+otherPage.ID, nil)
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandlers_Update_SavesDoc(t *testing.T) {
	pool := testPool(t)
	callerID, denID := testDen(t)
	created, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := sessionRequest(t, pool, callerID, http.MethodPatch, "/pages/"+created.ID, map[string]any{
		"doc": map[string]any{"type": "doc"},
	})
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	p, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !docEqual(t, p.Doc, `{"type":"doc"}`) {
		t.Errorf("doc = %s, want the saved document", p.Doc)
	}
}

func TestHandlers_Update_MalformedDoc_RejectedWithoutWriting(t *testing.T) {
	pool := testPool(t)
	callerID, denID := testDen(t)
	created, err := Create(t.Context(), pool, denID, "", "My Page")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rawToken, err := auth.CreateSession(t.Context(), pool, callerID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/pages/"+created.ID, bytes.NewReader([]byte(`{"doc": {not valid json}}`)))
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: rawToken})
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	p, err := Get(t.Context(), pool, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(p.Doc) != "{}" {
		t.Errorf("doc = %s, want unchanged empty document", p.Doc)
	}
}

func TestHandlers_List_ScopedToCallersOwnDen(t *testing.T) {
	pool := testPool(t)
	callerID, callerDenID := testDen(t)
	if _, err := Create(t.Context(), pool, callerDenID, "", "Mine"); err != nil {
		t.Fatalf("create own page: %v", err)
	}
	_, otherDenID := testDen(t)
	if _, err := Create(t.Context(), pool, otherDenID, "", "Not mine"); err != nil {
		t.Fatalf("create other page: %v", err)
	}

	req := sessionRequest(t, pool, callerID, http.MethodGet, "/pages", nil)
	rec := httptest.NewRecorder()
	newMux(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var nodes []Node
	if err := json.Unmarshal(rec.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Title != "Mine" {
		t.Errorf("got %+v, want exactly the caller's own page", nodes)
	}
}
