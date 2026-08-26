package auth

import (
	"encoding/json"
	"net/http"
)

// Config reports which sign-in methods are available, so the frontend can
// hide the Google button when no Google OAuth client is configured. This
// never mutates anything, so it's safe to expose as an unauthenticated GET.
func Config(googleEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"googleEnabled": googleEnabled})
	}
}
