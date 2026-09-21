## Why

Three gaps on the authentication surface, none of them a bug in what the code does, all of them things the code does not do at all.

**Nothing limits how many times an account's password can be guessed.** `POST /auth/login` is unauthenticated, publicly reachable, and accepts unlimited attempts. Signup is public, so the email half of a credential pair is not secret. A search of `server/` for any rate limiting, throttling, or attempt tracking returns nothing.

**The timing defence makes every login attempt expensive on purpose.** `password.go` compares against a fixed `dummyHash` when the email is unknown, so that an unknown email and a wrong password take the same time — correct, and it prevents account enumeration. The consequence is that no login attempt can be rejected cheaply: every one costs a full bcrypt round at `DefaultCost`, including attempts with addresses that obviously match nothing. On a single container, that turns an unauthenticated endpoint into a CPU amplifier. The answer is not to weaken the timing defence — it is the rate limit that is missing.

**Session rows accumulate forever.** `RevokeSession` runs only on an explicit sign-out. Every session that ends by closing the tab leaves a row behind permanently. `ResolveSession` filters on `expires_at > now()`, so nothing incorrect happens; the table simply never shrinks, and it holds one row per sign-in for the life of the deployment. Retaining a record of every session anyone has ever opened, indefinitely, is also more than the system needs to keep.

Separately, an audit that should have run has not. Task 13.4 of `archive/2026-08-26-add-den-workspace` — *"Audit every route and verify no `GET` endpoint changes state, as the CSRF posture depends on it"* — was archived unchecked. Running it now: every mutation is `POST`/`PATCH`/`DELETE` and the page handlers are clean, with one exception. `GET /auth/google/callback` creates a user and a session. It is defended, by the `__Host-` state cookie checked before the code exchange, and it is unavoidable, because OAuth redirects are GETs. But the rule as written admits no exception, so the audit cannot be honestly ticked, and the next person to run it stops in the same place.

## What Changes

- Limit repeated authentication attempts, so an account's password cannot be guessed without bound and an unauthenticated caller cannot force unlimited bcrypt work.
- Remove expired sessions rather than retaining them indefinitely.
- State the CSRF posture and its one exception as a requirement, including why the exception is safe, so the audit has something to be checked against. No behaviour changes here — this is writing down a property the system already has and a defence it already implements.

Not changing: the `dummyHash` timing path, `SameSite=Lax`, the CORS configuration, session lifetime, or the responses any auth endpoint returns on success or failure.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accounts`: adds a requirement limiting repeated authentication attempts; adds a requirement that expired sessions are removed rather than retained; adds a requirement stating that state-changing operations are not reachable by `GET`, with the provider callback named as the sole exception and its protection specified.

## Impact

- **`server/internal/auth`.** A limiter in front of `Login` and `Register`, and a periodic removal of expired session rows. Both are new code; neither changes an existing path.
- **Rate-limit state has to live somewhere.** In-process is simplest and correct for the one container running today; Postgres survives restarts and would still be correct if a second instance ever appears. The choice is a design decision, not a foregone one, and `design.md` settles it.
- **Client IP behind a proxy.** Railway, and any reverse proxy on the self-hosted target, terminate the connection, so `RemoteAddr` is the proxy. Anything keyed on IP needs a deliberate answer about forwarded headers — and a wrong answer makes the limit either useless or trivially bypassed.
- **`web/`.** A rejected attempt needs a message distinguishable from "wrong password"; `SignedOutLanding` already renders whatever the API returns, so this may be limited to the API's response text.
- **The archived change's task 13.4** is answerable once the requirement exists. This change does not reopen the archived change; the audit's outcome lands as a requirement here.
- **No impact** on the database schema beyond what rate-limit storage may need, on session issuance, or on the page API.
