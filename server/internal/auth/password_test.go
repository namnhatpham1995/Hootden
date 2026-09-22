package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	return PasswordHandlers{
		Pool:         pool,
		Store:        fakePasswordStore{pool: pool},
		CookieDomain: testCookieDomain,
		Limiter:      NewLoginAttemptLimiter(),
	}
}

// doJSONAddrCounter gives each doJSON call its own RemoteAddr. Tests in this
// file share a package-level handler's Limiter across many unrelated calls;
// without this, they'd all look like the same client and could trip on each
// other. A test that specifically wants to simulate one repeat client (the
// limiter tests) reuses the same email across calls instead, which is
// charged independently of the address -- see AttemptLimiter.AllowAttempt.
var doJSONAddrCounter int32

func doJSON(h http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	n := atomic.AddInt32(&doJSONAddrCounter, 1)
	req.RemoteAddr = fmt.Sprintf("192.0.2.%d:%d", 1+(n%253), 1024+n)
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
	var durations []time.Duration
	for name, req := range cases {
		start := time.Now()
		rec := doJSON(h.Login, http.MethodPost, "/auth/login", req)
		durations = append(durations, time.Since(start))
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

	// The limit is not an existence oracle either: every rejection path runs
	// the same bcrypt comparison (see dummyHash), so none should be a lot
	// faster or slower than the others. A generous tolerance keeps this from
	// being flaky on a loaded CI box while still catching a path that skips
	// the comparison outright.
	minD, maxD := durations[0], durations[0]
	for _, d := range durations[1:] {
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
	}
	if spread := maxD - minD; spread > 300*time.Millisecond {
		t.Errorf("rejection timing spread = %v, want under 300ms (durations: %v)", spread, durations)
	}
}

// spyPasswordStore counts FindUserByEmail calls so a test can assert the
// store was never touched -- and therefore bcrypt never ran, since Login
// only calls CompareHashAndPassword after a store lookup.
type spyPasswordStore struct {
	fakePasswordStore
	findCalls int
}

func (s *spyPasswordStore) FindUserByEmail(ctx context.Context, email string) (string, string, error) {
	s.findCalls++
	return s.fakePasswordStore.FindUserByEmail(ctx, email)
}

func TestLogin_RefusedAttemptSkipsBcrypt(t *testing.T) {
	pool := testPool(t)
	store := &spyPasswordStore{fakePasswordStore: fakePasswordStore{pool: pool}}
	h := PasswordHandlers{
		Pool:         pool,
		Store:        store,
		CookieDomain: testCookieDomain,
		Limiter:      NewAttemptLimiter(1, time.Hour, 10),
	}
	email := randomEmail(t)

	doJSON(h.Login, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "whatever"})
	if store.findCalls != 1 {
		t.Fatalf("findCalls after first attempt = %d, want 1", store.findCalls)
	}

	// Same email, second attempt: the limiter must refuse before the store
	// (and so bcrypt) is ever touched again.
	rec := doJSON(h.Login, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "whatever"})

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header")
	}
	if store.findCalls != 1 {
		t.Errorf("findCalls after refused attempt = %d, want still 1 (no credential evaluated)", store.findCalls)
	}
}

func TestRegister_RefusedByLimiter(t *testing.T) {
	pool := testPool(t)
	h := PasswordHandlers{
		Pool:         pool,
		Store:        fakePasswordStore{pool: pool},
		CookieDomain: testCookieDomain,
		Limiter:      NewAttemptLimiter(1, time.Hour, 10),
	}
	email := randomEmail(t)

	doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})

	rec := doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "another-password"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header")
	}
}

// TestLogin_MistypesThenSucceedsNeverRefused pins the threshold to the
// requirement it was chosen from: a person getting their own password wrong
// a handful of times before succeeding must never be caught by the limiter.
func TestLogin_MistypesThenSucceedsNeverRefused(t *testing.T) {
	pool := testPool(t)
	h := newTestPasswordHandlers(pool)
	email := randomEmail(t)
	doJSON(h.Register, http.MethodPost, "/auth/register", registerRequest{Email: email, Password: "correct-horse"})

	for i := 0; i < 4; i++ {
		rec := doJSON(h.Login, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "wrong-password"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("mistype %d: status = %d, want %d, body=%s", i, rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
	}

	rec := doJSON(h.Login, http.MethodPost, "/auth/login", loginRequest{Email: email, Password: "correct-horse"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("final correct attempt: status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
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
