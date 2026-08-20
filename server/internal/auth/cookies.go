package auth

import (
	"net/http"
	"time"
)

// __Host- forbids a Domain attribute, which correctly confines these
// short-lived flow cookies to this exact host regardless of CookieDomain.
const (
	stateCookieName    = "__Host-oauth-state"
	verifierCookieName = "__Host-oauth-verifier"
	flowCookieTTL      = 5 * time.Minute
)

// SessionCookieName is the long-lived session cookie, shared across the
// apex and api. subdomain via Domain, so it cannot use the __Host- prefix.
const SessionCookieName = "session"

func setFlowCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(flowCookieTTL.Seconds()),
	})
}

func clearFlowCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func readFlowCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

func setSessionCookie(w http.ResponseWriter, cookieDomain, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Domain:   cookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, cookieDomain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Domain:   cookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
