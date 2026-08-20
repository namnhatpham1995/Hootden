package httpapi

import "net/http"

// CORS allows credentialed cross-origin requests from exactly appOrigin.
// It never reflects an arbitrary Origin and never sends "*" -- both are
// incompatible with Access-Control-Allow-Credentials.
func CORS(appOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")

		if origin == appOrigin {
			w.Header().Set("Access-Control-Allow-Origin", appOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
