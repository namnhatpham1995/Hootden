package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfig_ReflectsGoogleEnabled(t *testing.T) {
	for _, want := range []bool{true, false} {
		req := httptest.NewRequest(http.MethodGet, "/auth/config", nil)
		rec := httptest.NewRecorder()
		Config(want)(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("googleEnabled=%v: status = %d, want %d", want, rec.Code, http.StatusOK)
		}
		var body struct {
			GoogleEnabled bool `json:"googleEnabled"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.GoogleEnabled != want {
			t.Errorf("googleEnabled = %v, want %v", body.GoogleEnabled, want)
		}
	}
}
