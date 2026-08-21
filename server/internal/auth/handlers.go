package auth

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
)

type Handlers struct {
	Pool         *pgxpool.Pool
	OAuthConfig  *oauth2.Config
	Exchanger    Exchanger
	AppOrigin    string
	CookieDomain string
}

// Start begins the Google sign-in flow: a fresh state and PKCE verifier are
// stashed in short-lived cookies, then the browser is redirected to Google.
func (h Handlers) Start(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to start sign-in")
		return
	}
	verifier := oauth2.GenerateVerifier()

	setFlowCookie(w, stateCookieName, state)
	setFlowCookie(w, verifierCookieName, verifier)

	authURL := h.OAuthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback verifies state before doing anything else. A missing, mismatched,
// or expired state cookie is rejected without ever attempting the code
// exchange, so no user or session is created for a request that didn't
// originate from Start.
func (h Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	defer clearFlowCookie(w, stateCookieName)
	defer clearFlowCookie(w, verifierCookieName)

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, h.AppOrigin+"/?auth=declined", http.StatusFound)
		return
	}

	wantState, ok := readFlowCookie(r, stateCookieName)
	gotState := r.URL.Query().Get("state")
	if !ok || wantState == "" || gotState == "" || wantState != gotState {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid or expired sign-in attempt")
		return
	}

	verifier, ok := readFlowCookie(r, verifierCookieName)
	if !ok || verifier == "" {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid or expired sign-in attempt")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	sub, email, err := h.Exchanger.Exchange(r.Context(), code, verifier)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusBadGateway, "sign-in failed")
		return
	}

	userID, _, err := GetOrCreateUserBySub(r.Context(), h.Pool, sub, email)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "sign-in failed")
		return
	}

	rawToken, err := CreateSession(r.Context(), h.Pool, userID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "sign-in failed")
		return
	}

	setSessionCookie(w, h.CookieDomain, rawToken, SessionTTL)
	http.Redirect(w, r, h.AppOrigin+"/", http.StatusFound)
}

// SignOut revokes the session server-side, not just the cookie, so the same
// token is rejected on any later request even if the cookie survives.
func (h Handlers) SignOut(w http.ResponseWriter, r *http.Request) {
	if rawToken, ok := readFlowCookie(r, SessionCookieName); ok && rawToken != "" {
		_ = RevokeSession(r.Context(), h.Pool, rawToken)
	}
	clearSessionCookie(w, h.CookieDomain)
	w.WriteHeader(http.StatusNoContent)
}
