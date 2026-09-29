package httpapi

import (
	"net"
	"net/http"
	"strings"
)

// ClientAddr identifies the address a request came from, for keying things
// like a rate limiter. trustedHops is how many reverse proxies sit in front
// of the server, each appending its own view of the peer to
// X-Forwarded-For: the client is the entry trustedHops from the end, since
// everything to its left arrived from the client and can be forged. 0 means
// no proxy -- the header is ignored and the TCP peer is used.
//
// A request carrying fewer entries than trustedHops skipped a proxy (e.g.
// it hit Railway directly instead of via Vercel), so the leftmost entry --
// written by the first proxy it did pass -- is used instead.
//
// ponytail: trusts that every request came through all trustedHops proxies.
// Production runs Vercel -> Railway (hops 2) but Railway's public hostname
// is also reachable directly, and there a client-supplied entry lands where
// Vercel's is expected, so a direct caller can pick its own address key.
// The per-email key still bounds guessing against any one account. Upgrade
// by having the Vercel hop attach a shared-secret header and trusting the
// second entry only when it's present.
func ClientAddr(r *http.Request, trustedHops int) string {
	xff := r.Header.Get("X-Forwarded-For")
	if trustedHops <= 0 || xff == "" {
		return remoteAddrHost(r.RemoteAddr)
	}
	entries := strings.Split(xff, ",")
	i := max(len(entries)-trustedHops, 0)
	if addr := strings.TrimSpace(entries[i]); addr != "" {
		return addr
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
