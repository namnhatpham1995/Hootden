## Context

See `proposal.md — Why`. What constrains the approach:

- One container per deploy target today. `archive/2026-08-26-add-den-workspace/design.md` states the property deliberately — "nothing session-related lives in process memory, so no deploy target is special-cased" — and immediately scopes it: "only true because there is no real-time transport in this version."
- Minimum dependency surface is a stated goal of the original design. The Go side currently depends on `pgx`, `goose`, `x/crypto`, and `x/oauth2`, and nothing else.
- Every deploy target sits behind something that terminates the connection: Railway's edge on the managed path, a reverse proxy on the self-hosted one. `RemoteAddr` is never the client.
- The timing-equality property in `Login` is not negotiable. Anything added in front of it must not reintroduce a way to tell an existing account from an absent one.

## Goals / Non-Goals

**Goals:**

- Password guessing is bounded, and a single caller cannot compel unlimited bcrypt work.
- The `GET`-mutation rule becomes something a test enforces rather than a person remembers to audit.

**Non-Goals:**

- No account lockout. A refusal that persists until an administrator lifts it turns a rate limit into a denial-of-service against the account's owner, and there is no administrator.
- No CAPTCHA, no proof-of-work, no second factor.
- No general-purpose rate limiting for the page API. Those routes are authenticated and bounded by a session; the unauthenticated endpoints are the exposure.
- No audit logging of failed attempts. Worth having eventually, a different change.

## Decisions

### In-process rate-limit state, with the ceiling written down

Two places the counters could live:

**Postgres** survives restarts and would still be correct with a second instance. But it makes every authentication attempt a write, which means an attacker floods the database instead of the CPU. Spending a row-write to avoid a bcrypt round is a poor trade, and it moves the load onto the one component that is hardest to scale.

**In process**, a map behind a mutex with a periodic sweep, is a few lines of standard library, adds no dependency, and evaluates in nanoseconds. It loses its counters on restart and does not aggregate across instances.

In-process, for now. Both weaknesses are bounded: a restart is not something an attacker can induce, and there is exactly one instance. The moment a second instance exists this becomes a per-instance limit, which is why it gets a `ponytail:` comment naming the ceiling and the upgrade path rather than being quietly assumed.

This does depart from the original design's "nothing in process memory" property, and the departure is worth being explicit about: that property exists so no deploy target is special-cased and restarts are safe. A lost rate-limit counter fails safe — it forgets that someone was being throttled, it never grants access — so neither guarantee is weakened.

**Alternative rejected — `golang.org/x/time/rate`.** A correct token bucket from a first-party module. Rejected because a fixed-window counter is around fifteen lines here and the module would be a new dependency against a stated minimum-dependency goal. If the limiting logic grows past what is obviously correct by reading it, take the dependency.

### Two keys: the account and the caller

Keyed on the submitted email alone, an attacker spreads attempts across many addresses and compels unlimited bcrypt work. Keyed on the caller alone, a botnet guesses one account's password freely. Neither key subsumes the other, so both are counted, and either can refuse.

The email key must count attempts for addresses with no account exactly as it counts real ones. Skipping the count for an unknown address would make the limit itself an oracle, undoing the `dummyHash` work.

### The client address is the last `X-Forwarded-For` entry, not the first

The first entry is whatever the client claimed and is trivially forged: sending `X-Forwarded-For: 1.2.3.4` would let an attacker present a new identity per request and bypass the caller limit entirely.

Each proxy *appends* the address it actually observed, so with exactly one trusted proxy in front the last entry is the real peer as the edge saw it, and a forged prefix survives only as an ignored earlier entry. Absent the header, `RemoteAddr` is used.

This is correct for exactly one trusted proxy, which is every current target. Two chained proxies would make the last entry the first proxy rather than the client, collapsing all traffic onto one key — it fails toward over-limiting rather than under-limiting, but it is a ceiling worth marking.

**Alternative rejected — a configurable count of trusted proxies.** The general solution, and configuration nobody would tune correctly for a deployment shape that is the same everywhere today.

### Reaping runs on a ticker in the server process

`DELETE FROM sessions WHERE expires_at < now()`, on a long interval, in a goroutine started at boot. The table is small, the statement is indexed by nothing in particular but scans a small table, and nothing depends on its timeliness — an expired session is already refused by `ResolveSession`, so reaping is purely about not retaining records.

**Alternative rejected — delete opportunistically during `ResolveSession`.** No goroutine and no ticker, but it adds a write to the hot path of every authenticated request to avoid a periodic one. Backwards.

**Alternative rejected — `pg_cron`.** Correct and out-of-process, but an extension the self-hosted and personal-machine targets would each have to install, to run one statement.

### The `GET`-mutation rule becomes a test

The audit that motivated this went unrun for the whole life of the change, and a rule enforced by remembering to audit is the kind that stays green until it does not.

Registering routes from a table instead of a sequence of `mux.Handle` calls makes the rule checkable: a test walks the table and asserts every `GET` entry is on an explicit read-only allowlist, with the provider callback named as the one exception. Adding a state-changing `GET` then fails the build, and adding a legitimate new exception forces someone to write it into the allowlist where the next reader will see it.

This is the only part of this change that touches existing code, and it is a mechanical restructuring of `main.go` with no behavioural difference.

## Risks / Trade-offs

**A rate limit can lock a legitimate person out of their own account** → Which is why the refusal is temporary, self-lifting, and needs no administrator. The threshold has to sit well above ordinary mistyping; the cost of setting it too high is small, and the cost of setting it too low lands on the one user who is not attacking anything.

**Counters reset on deploy** → A restart clears the state, so a throttled attacker gets a fresh budget. On a platform that redeploys on push, this is not rare. It fails safe, and it is the accepted cost of not writing to Postgres per attempt.

**The limiter becomes its own memory-growth vector** → An attacker naming a new email per request creates a new map entry per request. The sweep must evict expired windows on a schedule that does not depend on the attacker cooperating, and the map needs a bound past which new keys are refused rather than allocated.

**Rate limiting can leak account existence if done carelessly** → Any difference in threshold, timing, or message between a real and an absent address undoes `dummyHash`. The spec requires them indistinguishable; the tests have to assert it rather than assume it.

**Restructuring `main.go`'s routing to enable the test** → Touches every route registration at once, in a change otherwise about auth. Mitigated by it being its own task group and its own PR, verifiable by the existing suite passing unchanged.
