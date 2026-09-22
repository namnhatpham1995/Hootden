package main

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
	"github.com/namnhatpham1995/Hootden/server/internal/page"
	"github.com/namnhatpham1995/Hootden/server/internal/workspace"
)

type route struct {
	Method  string
	Path    string
	Handler http.Handler
}

// getMutationException is the one GET route allowed to change state: the
// OAuth provider's redirect target, which must be GET because the provider
// chooses the method. The protection this exception rests on is Callback's
// own state-cookie check (see auth.Handlers.Callback), which runs before any
// code exchange or account creation -- a request that didn't originate from
// Start is rejected before it can do anything.
const getMutationException = "/auth/google/callback"

// readOnlyGetPaths is the explicit allowlist every other GET route must
// appear on. Closes task 13.4 of archive/2026-08-26-add-den-workspace by
// making the audit a test instead of something remembered.
var readOnlyGetPaths = map[string]bool{
	"/healthz":           true,
	"/auth/config":       true,
	"/auth/google/start": true,
	"/me":                true,
	"/pages":             true,
	"/pages/{id}":        true,
}

// buildRoutes is the single source of truth for both the live server's
// routing table and the GET-mutation-rule test, so the two can't drift apart.
func buildRoutes(
	pool *pgxpool.Pool,
	authHandlers auth.Handlers,
	passwordHandlers auth.PasswordHandlers,
	pageHandlers page.Handlers,
	googleEnabled bool,
	cookieDomain string,
) []route {
	requireAuth := auth.RequireAuth(pool, cookieDomain)

	return []route{
		{"GET", "/healthz", httpapi.Healthz(pool)},
		{"GET", "/auth/config", auth.Config(googleEnabled)},
		{"GET", "/auth/google/start", http.HandlerFunc(authHandlers.Start)},
		{"GET", "/auth/google/callback", http.HandlerFunc(authHandlers.Callback)},
		{"POST", "/auth/register", http.HandlerFunc(passwordHandlers.Register)},
		{"POST", "/auth/login", http.HandlerFunc(passwordHandlers.Login)},
		{"POST", "/auth/signout", http.HandlerFunc(authHandlers.SignOut)},
		{"GET", "/me", requireAuth(workspace.Me(pool))},
		{"GET", "/pages", requireAuth(http.HandlerFunc(pageHandlers.List))},
		{"POST", "/pages", requireAuth(http.HandlerFunc(pageHandlers.Create))},
		{"GET", "/pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Get))},
		{"PATCH", "/pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Update))},
		{"DELETE", "/pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Delete))},
	}
}

// checkGetRoutesAreReadOnly fails if any GET route isn't on the allowlist
// and isn't the named exception, so a state-changing handler registered as
// GET breaks the build instead of the CSRF posture.
func checkGetRoutesAreReadOnly(routes []route, allowlist map[string]bool, exception string) error {
	for _, rt := range routes {
		if rt.Method != http.MethodGet || rt.Path == exception || allowlist[rt.Path] {
			continue
		}
		return fmt.Errorf("GET %s is not on the read-only allowlist and is not the named exception (%s)", rt.Path, exception)
	}
	return nil
}
