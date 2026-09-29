package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientAddr(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		setXFF     bool
		remoteAddr string
		want       string
	}{
		{
			name:       "forged single-value header",
			xff:        "203.0.113.99",
			setXFF:     true,
			remoteAddr: "10.0.0.5:54321",
			want:       "203.0.113.99",
		},
		{
			name:       "genuine one-proxy header",
			xff:        "198.51.100.23",
			setXFF:     true,
			remoteAddr: "10.0.0.1:443",
			want:       "198.51.100.23",
		},
		{
			name:       "multi-entry header takes the last hop",
			xff:        "1.2.3.4, 198.51.100.23",
			setXFF:     true,
			remoteAddr: "10.0.0.1:443",
			want:       "198.51.100.23",
		},
		{
			name:       "malformed header falls back to RemoteAddr",
			xff:        "198.51.100.23, ",
			setXFF:     true,
			remoteAddr: "10.0.0.1:443",
			want:       "10.0.0.1",
		},
		{
			name:       "no header falls back to RemoteAddr",
			setXFF:     false,
			remoteAddr: "192.0.2.7:8080",
			want:       "192.0.2.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.setXFF {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			got := ClientAddr(req)
			if got != tt.want {
				t.Errorf("ClientAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoteAddrHost_NoPort(t *testing.T) {
	// A RemoteAddr without a port (unusual, but SplitHostPort errors on it)
	// should pass through unchanged rather than being dropped.
	got := remoteAddrHost("192.0.2.7")
	if got != "192.0.2.7" {
		t.Errorf("remoteAddrHost() = %q, want %q", got, "192.0.2.7")
	}
}
