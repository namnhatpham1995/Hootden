package httpapi

import (
	"net"
	"net/http"
	"strings"
)

// ClientAddr identifies the address a request came from, for keying things
// like a rate limiter.
//
// ponytail: trusts exactly one reverse proxy in front of the server -- the
// last X-Forwarded-For entry is taken as that proxy's own view of the
// client, since only the proxy (not the client) can append to the header on
// its way in. A second chained proxy breaks this: the client's forged
// last-entry would then land where the trusted proxy is expected to write,
// and this function would trust it. Upgrade by validating the header against
// a configured number of trusted hops (or a trusted CIDR for the immediate
// peer) before taking the value.
func ClientAddr(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		entries := strings.Split(xff, ",")
		if last := strings.TrimSpace(entries[len(entries)-1]); last != "" {
			return last
		}
	}
	return remoteAddrHost(r.RemoteAddr)
}

func remoteAddrHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
