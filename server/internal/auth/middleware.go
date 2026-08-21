package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
)

type contextKey int

const userIDKey contextKey = 0

// UserID returns the signed-in user's ID from a request context populated
// by RequireAuth. Only call it on routes behind RequireAuth.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// RequireAuth rejects any request without a valid, unexpired session.
// Resolution always goes through ResolveSession against server-held state --
// the cookie's contents alone are never trusted. A stale (expired or
// unknown) session cookie is cleared rather than left for the browser to
// keep resending.
func RequireAuth(pool *pgxpool.Pool, cookieDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, ok := readFlowCookie(r, SessionCookieName)
			if !ok || rawToken == "" {
				httpapi.WriteJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			userID, err := ResolveSession(r.Context(), pool, rawToken)
			if errors.Is(err, ErrSessionNotFound) {
				clearSessionCookie(w, cookieDomain)
				httpapi.WriteJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if err != nil {
				httpapi.WriteJSONError(w, http.StatusInternalServerError, "authentication check failed")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
