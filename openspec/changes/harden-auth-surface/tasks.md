## 1. Session reaping

Ordered first because it is the smallest, touches nothing else, and depends on nothing below it.

- [ ] 1.1 Add a periodic removal of expired session rows, started at boot and running on a long interval, and verify with an integration test that a row past `expires_at` is gone after a cycle while an unexpired one remains
- [ ] 1.2 Verify reaping does not gate expiry — an expired session whose row has not yet been removed is still refused by `ResolveSession` — by asserting the existing expired-session test passes with reaping disabled
- [ ] 1.3 Verify the reaper stops cleanly on shutdown and logs a failed cycle without terminating the process, since a transient database error must not take the server down

## 2. Client address resolution

Small, independently testable, and needed by group 3. Split out so its edge cases get reviewed on their own rather than inside the limiter.

- [ ] 2.1 Add client-address resolution taking the last `X-Forwarded-For` entry, falling back to `RemoteAddr`, and verify table tests cover a forged single-value header, a genuine one-proxy header, a multi-entry header, a malformed header, and no header at all
- [ ] 2.2 Add a `ponytail:` comment naming the one-trusted-proxy ceiling and the upgrade path, and verify it states what breaks with two chained proxies

## 3. Attempt limiting

The substance of this change.

- [ ] 3.1 Add an in-process fixed-window limiter keyed independently on submitted email and on client address, with a bounded key count and a sweep that evicts expired windows, and verify unit tests cover the window opening, refusing at the threshold, and lifting on its own
- [ ] 3.2 Verify the limiter cannot grow without bound by driving it with a distinct key per request and confirming eviction holds the key count under its cap and that new keys past the cap are refused rather than allocated
- [ ] 3.3 Apply the limiter to `POST /auth/login` and `POST /auth/register`, refusing with `429` and a `Retry-After` before the credential is evaluated, and verify no bcrypt comparison runs for a refused attempt
- [ ] 3.4 Verify the refusal is indistinguishable between an email with an account and one without — same status, same body, same timing within tolerance — so the limit is not an existence oracle, and verify the existing "unknown email is rejected the same way" test still passes
- [ ] 3.5 Verify a person who mistypes a small number of times and then succeeds is never refused, choosing the threshold from that requirement rather than from the attack side
- [ ] 3.6 Add a `ponytail:` comment on the limiter naming the in-process ceiling — counters lost on restart, per-instance if a second ever runs — and the Postgres-backed upgrade path
- [ ] 3.7 Surface the refusal in `SignedOutLanding` as its own message rather than as a failed credential, and verify the person is told they have attempted too many times and can try again later

## 4. The GET-mutation rule becomes enforceable

Restructuring, no behaviour change. Its own PR so the existing suite passing is the whole review.

- [ ] 4.1 Register routes from a table in `main.go` instead of sequential `mux.Handle` calls, and verify the full server test suite and the Playwright suite pass unchanged
- [ ] 4.2 Add a test asserting every `GET` route is on an explicit read-only allowlist, with the provider callback named as the sole state-changing exception, and verify the test fails when a state-changing handler is registered as `GET`
- [ ] 4.3 Verify the callback's existing state-cookie check runs before any credential exchange or account creation, and record that as the protection the exception rests on, closing task 13.4 of `archive/2026-08-26-add-den-workspace`

## 5. Documentation

- [ ] 5.1 Document the rate limit in the README — that it exists, roughly what it bounds, and that it resets on restart — and verify a self-hoster reading it would not mistake a `429` during their own testing for a bug
- [ ] 5.2 Record the `GET`-mutation rule and its single exception in `AGENTS.md` or the server's own guidance, pointing at the test that enforces it, and verify the note explains why the exception is safe rather than only that it exists
