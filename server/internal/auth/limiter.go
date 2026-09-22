package auth

import (
	"sync"
	"time"
)

// loginAttemptLimit is sized off the UX requirement (design.md: "a person
// mistypes a small number of times"), not the attack side -- 20 clears
// ordinary mistyping with headroom to spare, and also happens to clear the
// password-auth e2e suite's real call volume (~8 calls, all sharing one
// client address since Chromium blocks script-set X-Forwarded-For; see that
// suite's own comment). If either requirement changes, re-derive from the
// UX case first.
const (
	loginAttemptLimit   = 20
	loginAttemptWindow  = 15 * time.Minute
	loginAttemptMaxKeys = 10_000
)

// AttemptLimiter is a fixed-window limiter for POST /auth/login and
// POST /auth/register, keyed independently on the submitted email and on the
// client address: a request is allowed only when both keys are still under
// their limit. Each key's window opens on its first hit and lifts on its own
// once it elapses -- there's no explicit reset.
//
// ponytail: counters live in process memory only. They're lost on restart
// (everyone's count silently resets to zero) and are per-instance if a
// second server instance ever runs (each instance enforces the limit
// independently, so the effective limit is the configured one times however
// many instances are behind the load balancer). Upgrade to a Postgres-backed
// counter if either matters.
type AttemptLimiter struct {
	limit   int
	window  time.Duration
	maxKeys int

	mu      sync.Mutex
	windows map[string]*attemptWindow
}

type attemptWindow struct {
	count   int
	resetAt time.Time
}

func NewAttemptLimiter(limit int, window time.Duration, maxKeys int) *AttemptLimiter {
	return &AttemptLimiter{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		windows: make(map[string]*attemptWindow),
	}
}

// NewLoginAttemptLimiter returns the limiter used for the real login and
// register handlers, sized so a person who mistypes their password a few
// times and then gets it right is never refused.
func NewLoginAttemptLimiter() *AttemptLimiter {
	return NewAttemptLimiter(loginAttemptLimit, loginAttemptWindow, loginAttemptMaxKeys)
}

// AllowAttempt reports whether an attempt from this email and client address
// may proceed. Both keys are always charged for the attempt -- neither is
// skipped because the other already refused -- so an attacker rotating one
// dimension (many emails from one address, or one email from many addresses)
// still runs into the other.
func (l *AttemptLimiter) AllowAttempt(email, clientAddr string) bool {
	emailOK := l.allow("email:" + email)
	addrOK := l.allow("addr:" + clientAddr)
	return emailOK && addrOK
}

// allow evicts expired windows, then checks and charges the given key
// against its own window. A brand-new key is refused rather than allocated
// once the map is already at maxKeys, so an attacker cycling through
// distinct keys can't grow it without bound.
func (l *AttemptLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.sweep(now)

	w, ok := l.windows[key]
	if !ok {
		if len(l.windows) >= l.maxKeys {
			return false
		}
		l.windows[key] = &attemptWindow{count: 1, resetAt: now.Add(l.window)}
		return true
	}

	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}

func (l *AttemptLimiter) sweep(now time.Time) {
	for key, w := range l.windows {
		if now.After(w.resetAt) {
			delete(l.windows, key)
		}
	}
}
