package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
)

// TestMe_ResolvesFromSessionCookieAlone drives Me through the real
// RequireAuth middleware with nothing but a session cookie -- no id is
// ever passed as input, matching the spec's "resolves from the session
// cookie alone".
func TestMe_ResolvesFromSessionCookieAlone(t *testing.T) {
	pool := testPool(t)
	email := randomEmail(t)
	userID, denID, _, err := EnsureUserAndDen(t.Context(), pool, randomSub(t), email)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	rawToken, err := auth.CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: rawToken})
	rec := httptest.NewRecorder()

	auth.RequireAuth(pool, "")(Me(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Workspace struct {
			ID string `json:"id"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.ID != userID {
		t.Errorf("id = %q, want %q", body.ID, userID)
	}
	if body.Email != email {
		t.Errorf("email = %q, want %q", body.Email, email)
	}
	if body.Workspace.ID != denID {
		t.Errorf("workspace.id = %q, want %q", body.Workspace.ID, denID)
	}
}

func TestMe_NoCookie_Unauthorized(t *testing.T) {
	pool := testPool(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	auth.RequireAuth(pool, "")(Me(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
