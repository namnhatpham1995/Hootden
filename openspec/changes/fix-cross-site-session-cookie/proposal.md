## Why

Sign-in is broken in production. Registering an account works — the row lands in Postgres — but the browser is returned to the signed-out landing screen instead of the Den, and signing in afterwards does the same.

The cause is not a bug in the sign-in code. The web app is deployed at `hootden.vercel.app` and the API at `hootden-production.up.railway.app`. Because `vercel.app` and `up.railway.app` are separate entries on the Public Suffix List, these are different registrable domains, so every request from the app to the API is **cross-site**. The session cookie is issued `Secure; SameSite=Lax`, which browsers refuse to store from a cross-site response and refuse to send on a cross-site request. Verified against the live deployment:

```
POST https://hootden-production.up.railway.app/auth/register
  Origin: https://hootden.vercel.app
→ HTTP/1.1 204 No Content
  Set-Cookie: session=…; Path=/; Max-Age=2592000; HttpOnly; Secure; SameSite=Lax
                                                  (no Domain — COOKIE_DOMAIN is empty)
→ the browser discards it, GET /me returns 401, the app renders the landing screen
```

The API is otherwise correctly configured: `APP_ORIGIN` matches the Vercel origin, CORS returns `Access-Control-Allow-Credentials: true`, and migrations have run. Only the cookie's site scope is wrong.

This deployment has never satisfied the accounts spec's existing requirement *"Session cookie is usable across the app and API origins"*, which already states the cookie must be scoped to a shared parent domain and must not require `SameSite=None`. `design.md` in `archive/2026-08-26-add-den-workspace` rejected this exact hostname pair by name, and `tasks.md` group 1 ("Domain and external setup", ordered first precisely because it is encoded into the cookie) was archived with 1.1–1.4 unchecked. The prerequisite was skipped, and nothing in the repository could detect it: `go test ./...` passes, the Playwright suite passes because localhost is single-site, and CI is green.

## What Changes

- Bind a registered domain: the apex to Vercel, an `api.` subdomain to Railway, replacing both platform-issued hostnames.
- Set the three deployment variables that encode it — `COOKIE_DOMAIN` to the shared parent, `APP_ORIGIN` to the apex origin, `NEXT_PUBLIC_API_ORIGIN` to the `api.` origin.
- Add a startup check that refuses to boot when `APP_ORIGIN` and `COOKIE_DOMAIN` describe a configuration whose session cookie cannot reach the app. This is the only behavioural change, and it is what stops the failure recurring: today a misconfigured deployment starts cleanly, serves healthy responses, and fails silently at the browser.
- Document that platform-issued hostnames cannot work and that no value of `COOKIE_DOMAIN` rescues them, since a public suffix is not a legal cookie domain.

Explicitly **not** changing: `setSessionCookie`'s attributes. `Secure; SameSite=Lax` is correct and is the CSRF posture the whole API depends on (`design.md`: "CSRF is covered by `SameSite=Lax` plus restrictive CORS"). Relaxing it to `SameSite=None` would make the session a third-party cookie, which Safari's Intelligent Tracking Prevention blocks outright — sign-in would keep working on desktop Chrome and fail on iPhone.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accounts`: adds a requirement that a deployment which cannot deliver its session cookie to the web app must fail at startup rather than at sign-in. The existing "Session cookie is usable across the app and API origins" requirement is unchanged — it is already correct, and this change makes a deployment that violates it detectable.

## Impact

- **Deployment (the substance of this change).** A registered domain, two DNS records, custom domains bound on both Vercel and Railway, and three environment variables. No application code depends on which hostnames are used.
- **`server/internal/config`.** `Load()` gains the coherence check and a new failure mode; `main.go` already treats a config error as fatal.
- **Google OAuth.** Currently unconfigured (`/auth/config` reports `googleEnabled: false`), so nothing to redo. If it is enabled later, the authorized redirect URI must be created against the final `api.<domain>` host — which is why the original task ordering put the domain ahead of the OAuth client.
- **Vercel preview deployments.** Each gets a fresh `*.vercel.app` URL, so previews remain cross-site and sign-in will not work on them regardless of this change. Production behind the custom domain is unaffected.
- **Existing sessions and accounts.** Session rows survive; their cookies do not, since the cookie's host changes. Everyone signs in again once. Accounts, Dens, and pages are untouched.
- **No impact** on `server/internal/auth`, the frontend, the database schema, or the test suites.
