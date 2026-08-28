package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// fakePasswordStore is a stand-in for workspace.PasswordStore, which this
// package must not import (see store.go).
type fakePasswordStore struct {
	pool *pgxpool.Pool
}

func (s fakePasswordStore) CreateUserWithPassword(ctx context.Context, email, passwordHash string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, passwordHash,
	).Scan(&userID)
	if err != nil {
		return "", ErrEmailTaken
	}
	return userID, nil
}

func (s fakePasswordStore) FindUserByEmail(ctx context.Context, email string) (string, string, error) {
	var userID, passwordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, COALESCE(password_hash, '') FROM users WHERE email = $1`, email,
	).Scan(&userID, &passwordHash)
	if err != nil {
		return "", "", ErrAccountNotFound
	}
	return userID, passwordHash, nil
}

func newTestPasswordHandlers(pool *pgxpool.Pool) PasswordHandlers {
	return PasswordHandlers{Pool: pool, Store: fakePasswordStore{pool: pool}, CookieDomain: testCookieDomain}
}

func doJSON(h http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName {
			return c.Value
		}
	}
	return ""
}

func TestRegister_Success(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)
	email := randomEmail(t)

	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	token := sessionCookie(rec)
	if token == "" {
		t.Fatal("expected a session cookie to be set")
	}
	if _, err := ResolveSession(t.Context(), pool, token); err != nil {
		t.Errorf("ResolveSession: %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)
	email := randomEmail(t)

	doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})
	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "another-password"})

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)

	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: randomEmail(t), Password: "short"})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegister_MalformedEmail(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)

	cases := []string{"", "no-at-sign.example.com", "user@"}
	for _, email := range cases {
		rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("email %q: status = %d, want %d", email, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestRegister_PasswordTooLong(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)

	tooLong := strings.Repeat("a", maxPasswordLength+1)
	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: randomEmail(t), Password: tooLong})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "72") {
		t.Errorf("body = %q, want it to name the limit", rec.Body.String())
	}
}

func TestRegister_PasswordAtMaxLength(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)

	atLimit := strings.Repeat("a", maxPasswordLength)
	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: randomEmail(t), Password: atLimit})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestLogin_CorrectCredentials(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)
	email := randomEmail(t)
	doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})

	rec := doJSON(h.Login, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "correct-horse"})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if sessionCookie(rec) == "" {
		t.Error("expected a session cookie to be set")
	}
}

// TestLogin_RejectedCasesLookIdentical confirms a wrong password, an unknown
// email, and a Google-only account with no password set all produce the
// exact same response -- none of the three should be distinguishable from
// the outside.
func TestLogin_RejectedCasesLookIdentical(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)

	registeredEmail := randomEmail(t)
	doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: registeredEmail, Password: "correct-horse"})

	googleOnlyEmail := randomEmail(t)
	_, err := pool.Exec(t.Context(),
		`INSERT INTO users (google_sub, email) VALUES ($1, $2)`,
		randomSub(t), googleOnlyEmail,
	)
	if err != nil {
		t.Fatalf("setup Google-only account: %v", err)
	}

	cases := map[string]loginRequest{
		"wrong password":       {Email: registeredEmail, Password: "totally-wrong"},
		"unknown email":        {Email: randomEmail(t), Password: "whatever"},
		"google-only, no pass": {Email: googleOnlyEmail, Password: "whatever"},
		"malformed email":      {Email: "not-an-email", Password: "whatever"},
		"over-long password":   {Email: registeredEmail, Password: strings.Repeat("a", maxPasswordLength+1)},
	}

	var bodies []string
	for name, req := range cases {
		rec := doJSON(h.Login, http.MethodPost, "/auth/login", req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want %d", name, rec.Code, http.StatusUnauthorized)
		}
		if sessionCookie(rec) != "" {
			t.Errorf("%s: expected no session cookie", name)
		}
		bodies = append(bodies, rec.Body.String())
	}
	for i := 1; i < len(bodies); i++ {
		if bodies[i] != bodies[0] {
			t.Errorf("response bodies differ across rejection cases: %q vs %q", bodies[0], bodies[i])
		}
	}
}

func randomSub(t *testing.T) string {
	t.Helper()
	tok, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	return tok
}

func randomEmail(t *testing.T) string {
	t.Helper()
	return randomSub(t) + "@example.com"
}
