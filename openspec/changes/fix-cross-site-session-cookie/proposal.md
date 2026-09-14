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

- Add a Next.js rewrite (`web/next.config.ts`) so the browser only ever talks to one origin — the Vercel app. Vercel forwards `/api/*` requests to Railway server-side; the frontend calls a relative `/api/...` path instead of an absolute, build-time-baked API origin.
- Set `API_ORIGIN` equal to `APP_ORIGIN` on Railway (both the Vercel origin) and leave `COOKIE_DOMAIN` empty — a host-only cookie, since as far as the browser can tell there is only one host. No code change to the startup check added earlier in this change (`server/internal/config`): it already accepts `APP_ORIGIN == API_ORIGIN` with an empty `COOKIE_DOMAIN`, which is exactly this shape.
- Rename `NEXT_PUBLIC_API_ORIGIN` to `API_PROXY_TARGET` and drop the `NEXT_PUBLIC_` prefix — the rewrite target is read server-side to build the proxy rule, never shipped to the browser.
- Document the proxy in the README's deploy section, replacing the domain-registration instructions.
- **No registered domain, no DNS, no custom domain bound on Vercel or Railway.** At the deployment owner's explicit direction this is a portfolio project, not worth a recurring registrar fee — this replaces group 1 of this change's original plan, which assumed a domain would be purchased.

Explicitly **not** changing: `setSessionCookie`'s attributes. `Secure; SameSite=Lax` is correct and is the CSRF posture the whole API depends on (`design.md`: "CSRF is covered by `SameSite=Lax` plus restrictive CORS"). Relaxing it to `SameSite=None` would make the session a third-party cookie, which Safari's Intelligent Tracking Prevention blocks outright — sign-in would keep working on desktop Chrome and fail on iPhone. This constraint is independent of the domain decision, so it still rules out `SameSite=None` here.

This also reverses a decision in the original design (`archive/2026-08-26-add-den-workspace/design.md`), which rejected proxying because "the usual reason to do it (hiding a cross-site cookie problem) does not exist once the domain decision above is made." That domain decision has now been made the other way, so the objection no longer applies.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accounts`: adds a requirement that a deployment which cannot deliver its session cookie to the web app must fail at startup rather than at sign-in. The existing "Session cookie is usable across the app and API origins" requirement is unchanged — it is already correct, and this change makes a deployment that violates it detectable.

## Impact

- **`web/next.config.ts`, `web/lib/api.ts`, `web/Dockerfile`, `web/.env.example`, `docker-compose.yml`.** The substance of this change now lives in the frontend, not deployment config alone: a rewrite rule, a relative-path API client, and a renamed build variable.
- **Deployment.** Two environment variables (`API_PROXY_TARGET` on Vercel, `API_ORIGIN` on Railway) — both platform-issued hostnames (`hootden.vercel.app`, the Railway one) stay exactly as they are today. No DNS, no custom domain, no certificate to manage on either platform.
- **`server/internal/config`.** No code change. The cookie-reachability check merged earlier in this change already accepts `APP_ORIGIN == API_ORIGIN` with `COOKIE_DOMAIN` empty; only the deployed value of `API_ORIGIN` changes.
- **Latency.** Every API request now makes one extra hop (browser → Vercel → Railway) instead of going straight to Railway. Vercel's rewrite proxying runs at its edge, not as a round trip back to a client; the added cost is real but small, and acceptable at this project's traffic level.
- **Google OAuth.** Currently unconfigured (`/auth/config` reports `googleEnabled: false`), so nothing to redo. If enabled later, the authorized redirect URI is created against the Vercel origin directly, since that's now the only origin the browser ever sees.
- **Vercel preview deployments.** Likely fixed as a side effect: each preview proxies `/api/*` to the same Railway backend under its own preview host, so a preview becomes same-origin too, unlike the domain plan which left previews permanently cross-site. Verified in task 1.4/5.3, not assumed.
- **Existing sessions and accounts.** Sign-in is already broken in production, so there is no working session to invalidate — this cutover has strictly less migration risk than the domain plan did. Accounts, Dens, and pages are untouched.
- **No impact** on `server/internal/auth`, the database schema, or the test suites.
