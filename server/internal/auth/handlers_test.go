package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// fakeExchanger never talks to Google. wantCalled asserts whether the
// handler under test was expected to reach the exchange step at all --
// state verification must reject bad requests before this point.
type fakeExchanger struct {
	t          *testing.T
	sub        string
	email      string
	err        error
	wantCalled bool
	called     bool
}

func (f *fakeExchanger) Exchange(ctx context.Context, code, codeVerifier string) (string, string, error) {
	f.called = true
	if !f.wantCalled {
		f.t.Error("Exchange was called but should not have been reached")
	}
	return f.sub, f.email, f.err
}

func newTestHandlers(pool *pgxpool.Pool, exch Exchanger) Handlers {
	return Handlers{
		Pool:         pool,
		OAuthConfig:  NewOAuthConfig("test-client-id", "test-client-secret", "https://api.hootden.example/auth/google/callback"),
		Exchanger:    exch,
		AppOrigin:    "https://hootden.example",
		CookieDomain: testCookieDomain,
	}
}

func TestStart_RedirectsWithStateAndPKCE(t *testing.T) {
	h := newTestHandlers(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/start", nil)
	rec := httptest.NewRecorder()
	h.Start(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	q := loc.Query()
	if q.Get("state") == "" {
		t.Error("redirect URL missing state")
	}
	if q.Get("code_challenge") == "" {
		t.Error("redirect URL missing code_challenge (PKCE)")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", q.Get("code_challenge_method"))
	}

	cookies := rec.Result().Cookies()
	var sawState, sawVerifier bool
	for _, c := range cookies {
		if c.Name == stateCookieName {
			sawState = true
			if c.Value != q.Get("state") {
				t.Error("state cookie does not match the state sent to Google")
			}
		}
		if c.Name == verifierCookieName {
			sawVerifier = true
		}
		if c.Name == stateCookieName || c.Name == verifierCookieName {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Errorf("cookie %q attributes = HttpOnly:%v Secure:%v SameSite:%v, want all set", c.Name, c.HttpOnly, c.Secure, c.SameSite)
			}
		}
	}
	if !sawState || !sawVerifier {
		t.Errorf("expected both state and verifier cookies to be set, got state=%v verifier=%v", sawState, sawVerifier)
	}
}

func TestCallback_MissingState_ExchangeNeverCalled(t *testing.T) {
	exch := &fakeExchanger{t: t, wantCalled: false}
	h := newTestHandlers(nil, exch)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCallback_MismatchedState_ExchangeNeverCalled(t *testing.T) {
	exch := &fakeExchanger{t: t, wantCalled: false}
	h := newTestHandlers(nil, exch)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=wrong&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "expected"})
	req.AddCookie(&http.Cookie{Name: verifierCookieName, Value: "verifier"})
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCallback_Declined_ExchangeNeverCalled(t *testing.T) {
	exch := &fakeExchanger{t: t, wantCalled: false}
	h := newTestHandlers(nil, exch)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?error=access_denied", nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "https://hootden.example/?auth=declined" {
		t.Errorf("Location = %q", loc)
	}
}

func TestCallback_Success_CreatesUserAndSession(t *testing.T) {
	pool := testPool(t)
	sub, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	exch := &fakeExchanger{t: t, wantCalled: true, sub: sub, email: "new@example.com"}
	h := newTestHandlers(pool, exch)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=s&code=c", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "s"})
	req.AddCookie(&http.Cookie{Name: verifierCookieName, Value: "v"})
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if !exch.called {
		t.Fatal("expected Exchange to be called for a valid state")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "https://hootden.example/" {
		t.Errorf("Location = %q", loc)
	}

	var sessionToken string
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName {
			sessionToken = c.Value
		}
	}
	if sessionToken == "" {
		t.Fatal("expected a session cookie to be set")
	}

	userID, err := ResolveSession(t.Context(), pool, sessionToken)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}

	var storedSub string
	if err := pool.QueryRow(t.Context(), `SELECT google_sub FROM users WHERE id = $1`, userID).Scan(&storedSub); err != nil {
		t.Fatalf("query created user: %v", err)
	}
	if storedSub != sub {
		t.Errorf("stored google_sub = %q, want %q", storedSub, sub)
	}
}

func TestSignOut_RevokesSession(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)
	rawToken, err := CreateSession(t.Context(), pool, userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	h := newTestHandlers(pool, nil)

	req := withSessionCookie(httptest.NewRequest(http.MethodPost, "/auth/signout", nil), rawToken)
	rec := httptest.NewRecorder()
	h.SignOut(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected the session cookie to be cleared")
	}

	_, err = ResolveSession(t.Context(), pool, rawToken)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("ResolveSession after sign-out = %v, want ErrSessionNotFound", err)
	}
}
