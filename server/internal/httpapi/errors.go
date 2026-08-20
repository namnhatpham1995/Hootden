package httpapi

import (
	"encoding/json"
	"net/http"
)

// WriteJSONError writes a JSON error body: {"error": message}.
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
