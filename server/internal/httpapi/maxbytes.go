package httpapi

import "net/http"

// MaxBodyBytes bounds every request body. Handlers that read r.Body get an
// error once this limit is exceeded, rather than the server buffering an
// unbounded body into memory.
const MaxBodyBytes = 1 << 20 // 1 MB

func MaxBytes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
