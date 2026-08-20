package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// randomToken returns 32 cryptographically random bytes, base64url-encoded.
// Used for both session tokens and the OAuth state parameter.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
