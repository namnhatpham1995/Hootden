package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Exchanger turns an OAuth authorization code into the signed-in person's
// stable identity. Swappable so tests never need a real Google account --
// only GoogleExchanger talks to Google.
type Exchanger interface {
	Exchange(ctx context.Context, code, codeVerifier string) (sub, email string, err error)
}

func NewOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"openid", "email"},
	}
}

type GoogleExchanger struct {
	OAuthConfig *oauth2.Config
}

func (g GoogleExchanger) Exchange(ctx context.Context, code, codeVerifier string) (sub, email string, err error) {
	token, err := g.OAuthConfig.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return "", "", fmt.Errorf("exchange code: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := g.OAuthConfig.Client(ctx, token).Do(req)
	if err != nil {
		return "", "", fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo returned status %d", resp.StatusCode)
	}

	var body struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", fmt.Errorf("decode userinfo: %w", err)
	}
	if body.Sub == "" {
		return "", "", fmt.Errorf("userinfo response missing sub")
	}
	return body.Sub, body.Email, nil
}
