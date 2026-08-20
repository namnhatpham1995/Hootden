package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSetSessionCookie_Attributes(t *testing.T) {
	rec := httptest.NewRecorder()
	setSessionCookie(rec, ".hootden.example", "raw-token-value", 30*24*time.Hour)

	header := rec.Header().Get("Set-Cookie")
	// Go's net/http strips a leading "." from Domain when writing the
	// header (RFC 6265 makes it optional -- Domain=hootden.example already
	// covers every subdomain, identically to the classic ".hootden.example"
	// form design.md describes).
	for _, want := range []string{
		"session=raw-token-value",
		"Domain=hootden.example",
		"HttpOnly",
		"Secure",
		"SameSite=Lax",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie = %q, missing %q", header, want)
		}
	}
}
