## Context

See `proposal.md — Why` for the failure and the evidence. The relevant constraints:

- `setSessionCookie` in `server/internal/auth/cookies.go` is correct and stays untouched. `Secure; SameSite=Lax` is the CSRF posture the whole API rests on, per `archive/2026-08-26-add-den-workspace/design.md` — "CSRF is covered by `SameSite=Lax` plus restrictive CORS", with the accompanying self-imposed rule that no `GET` endpoint changes state.
- The accounts spec already requires the cookie to work across the app and API origins. That requirement is not being changed; this change makes a deployment that violates it detectable.
- The same repository must run on three targets — Railway + Vercel, a rented VPS, and a personal machine — with only environment variables differing. Any check added here has to hold on all three, and must not break `localhost` development, where the app is on `:3000`, the API on `:8080`, and `COOKIE_DOMAIN` is deliberately empty.
- The server does not currently know its own public hostname. Railway terminates TLS and proxies, so the listening port says nothing about the origin a browser sees.
- The deployment owner has ruled out registering a domain for this project: "no need new domain, it is only a portfolio project." That removes the option group 1 of this change was originally built around, and is why the Decisions section below picks a different mechanism than the one group 1's tasks described.

## Goals / Non-Goals

**Goals:**

- Production sign-in works, using the cookie shape already specified.
- A deployment whose cookie cannot reach the app is impossible to leave running silently.
- The check is exact cookie semantics, not a heuristic that could reject a valid deployment.

**Non-Goals:**

- No change to the session mechanism, the cookie attributes, CORS, or the CSRF posture.
- No validation of anything else in the configuration. This is not a general config linter; it covers the one invariant whose violation is invisible until a user reports it.
- No attempt to make Vercel preview deployments work — though the chosen mechanism below is expected to fix them as a side effect; that is a bonus, not a goal driving the decision.
- No custom domain, DNS, or TLS certificate management. Ruled out by the deployment owner's explicit instruction, not a technical constraint.

## Decisions

### Proxy through Next.js rewrites; do not register a domain or relax the cookie

Three ways out of a cross-site session. The domain was the original choice; it is no longer available, which changes which of the remaining two wins:

| | Mechanism | Verdict |
|---|---|---|
| Register a domain, apex to Vercel, `api.` to Railway | Cookie stays `SameSite=Lax`, scoped to the shared parent | Rejected. Technically sound — zero application code, and it's the plan the original design already wrote down — but it costs a recurring registrar fee, and the deployment owner has ruled that out for what is a portfolio project. |
| Proxy the API through Next.js rewrites | The browser only ever talks to one origin; requests become same-origin | **Chosen.** Reverses the original design's rejection of this option, which reasoned that proxying only exists "to hide a cross-site cookie problem" that "does not exist once the domain decision above is made" (`archive/2026-08-26-add-den-workspace/design.md`). The domain decision was the premise; without it, the objection doesn't apply. Real costs remain — one extra hop per request, and a rewrite rule to keep in sync with the API's route surface — but they're smaller than a recurring domain cost, and the fix likely also covers Vercel preview deployments, which the domain plan left permanently broken. |
| `SameSite=None; Secure` | Cookie becomes third-party | Rejected, independently of the domain decision. Safari's ITP blocks third-party cookies outright, so sign-in would work on desktop Chrome and fail on iPhone — mobile is a stated constraint. Also discards the CSRF defence, which would then need double-submit tokens to replace. |

### The proxy makes `API_ORIGIN` equal to `APP_ORIGIN` in production

The browser only ever calls the Vercel origin — for the page and, through the rewrite, for the API. A cookie's `Set-Cookie` response is scoped by the browser to the host of the request it answered, not to whichever server actually generated the header; that's a property of how the browser attributes a response to its request, true for any transparent proxy. So the session cookie Railway issues, once it passes back through Vercel's rewrite, is scoped to the Vercel host exactly as if Vercel had issued it itself.

That means `API_ORIGIN` — "this API's own public origin, as a browser reaching it from `APP_ORIGIN` sees it" (`config.go`'s own doc comment) — is now, in production, the same value as `APP_ORIGIN`. `COOKIE_DOMAIN` stays empty: a host-only cookie is exactly right when there is only one host.

No change to `server/internal/config` is needed. The `checkCookieReachability` function added in group 2 of this change already has a branch for `COOKIE_DOMAIN` empty: `appHost == apiHost`. Setting both variables to the same value on Railway satisfies it directly — the same code path local development already exercises (`localhost` / `localhost` / empty), now also the production one.

### The rewrite target is a server-only variable, not `NEXT_PUBLIC_`

`web/next.config.ts`'s `rewrites()` destination is resolved when Next.js computes the build's routes manifest — the same build-time timing `NEXT_PUBLIC_API_ORIGIN` had, so changing the target still needs a redeploy, not just a variable change. What changes is that the value is no longer embedded in client-side JavaScript — previously every browser downloaded the Railway hostname in the bundle, an unnecessary exposure even though not sensitive — and the frontend's own code no longer needs to know any hostname at all, since it calls a same-origin relative path. `NEXT_PUBLIC_API_ORIGIN` is renamed to `API_PROXY_TARGET`; it stays a Docker build `ARG` in `web/Dockerfile` for the same reason as before, just serving the rewrite manifest instead of the client bundle.

### The startup check compares hosts, not sites

The tempting check is "are the app and API the same site?", which needs the Public Suffix List to answer correctly — a new dependency and an embedded list that goes stale.

It is not needed. Cookie delivery is decidable from the cookie's own rules, with no knowledge of public suffixes:

- `COOKIE_DOMAIN` empty: the cookie is host-only and reaches only the exact host that set it. So the app's host must **equal** the API's host.
- `COOKIE_DOMAIN` set: the cookie reaches that domain and its subdomains. So both the app's host and the API's host must be **equal to or a subdomain of** `COOKIE_DOMAIN`.

That is exact, not approximate. It accepts every valid deployment and rejects this one:

```
  app host                      api host                     COOKIE_DOMAIN     verdict
  ----------------------------  ---------------------------  ---------------   -------
  hootden.vercel.app            hootden-prod.up.railway.app  (empty)           REFUSE  <- today, direct calls
  hootden.vercel.app            hootden.vercel.app           (empty)           ok      <- target, via the proxy
  localhost                     localhost                    (empty)           ok      <- local dev
  hootden.example               api.hootden.example          hootden.example   ok      <- self-hosted, own domain
  hootden.example               api.hootden.example          (empty)           REFUSE
  hootden.example               api.other.example            hootden.example   REFUSE
```

**Alternative rejected — check at request time from the `Host` header.** It needs no new configuration and sees the real proxied host, but it can only fail a request, not a boot. An operator would learn about the misconfiguration from a 500 during sign-in, which is a smaller improvement over learning it from a user report than it looks. Fail-fast at startup is the point.

### `API_ORIGIN` becomes a required variable

The check needs the API's own public origin, and the server has no way to derive it. So it becomes explicit configuration alongside `APP_ORIGIN`, required on every target.

**Alternative rejected — derive it from `GOOGLE_REDIRECT_URL`.** That variable is optional (Google sign-in is currently off, and `/auth/config` reports `googleEnabled: false`), so the check would silently skip itself on exactly the password-only deployment that is broken today.

**Alternative rejected — read `RAILWAY_PUBLIC_DOMAIN`.** Couples the check to one platform, and would not fire on the VPS or personal-machine targets.

Making it required rather than optional is deliberate: an optional variable that disables a safety check when unset reproduces the failure this change exists to prevent.

## Risks / Trade-offs

**One extra network hop per request** → Every API call now goes browser → Vercel → Railway instead of browser → Railway directly. Vercel's rewrite proxying happens at its edge, not as a second client round trip; the added latency is real but small, and acceptable at this project's traffic level. Not mitigated further — it's the accepted cost of the chosen mechanism.

**The rewrite rule must stay in sync with the API's route surface** → A single catch-all (`/api/:path*` → `${API_PROXY_TARGET}/:path*`) covers every current and future route by pattern, not by listing them individually, so this only breaks if the frontend and the API disagree about the `/api` prefix convention itself — not per-route drift.

**A `COOKIE_DOMAIN` that is itself a public suffix passes the check but is rejected by browsers** → Still true, but no longer a production concern: production leaves `COOKIE_DOMAIN` empty under the proxy. It remains relevant only to the self-hosted target, where an operator sets a real domain directly (see the design's existing host-comparison table).

**The check cannot see a reverse proxy that rewrites the host** → On the self-hosted target, Caddy/nginx/Traefik sits in front and `API_ORIGIN` describes the public origin, not the container's — unchanged by this revision, since self-hosting isn't part of this pivot.

**No working session exists to lose** → Production sign-in is already broken, so unlike a typical cutover there's no session to invalidate. This makes the migration strictly lower-risk than the domain plan it replaces, which would have orphaned real cookies.

## Migration Plan

1. Add `async rewrites()` to `web/next.config.ts`, forwarding `/api/:path*` to `${API_PROXY_TARGET}/:path*` (defaulting to `http://localhost:8080` for local dev).
2. Change `web/lib/api.ts` to call relative `/api/...` paths instead of building an absolute URL from `API_ORIGIN`.
3. Rename `NEXT_PUBLIC_API_ORIGIN` to `API_PROXY_TARGET` in `web/Dockerfile`, `web/.env.example`, and `docker-compose.yml`'s `web` service.
4. Set `API_PROXY_TARGET` on Vercel to the Railway origin and redeploy — it's baked into the routes manifest at build time, so a variable change alone does nothing.
5. Set `API_ORIGIN` on Railway equal to `APP_ORIGIN`'s value (the Vercel origin), and leave `COOKIE_DOMAIN` empty.
6. Verify sign-in on the Vercel origin: `Set-Cookie` carries no `Domain` attribute, the browser stores it, `GET /api/me` returns 200 through the proxy, and the Den renders.
7. Delete the `diagnostic-probe-delete-me@example.invalid` account left by the investigation.

**Rollback:** revert the `next.config.ts`/`lib/api.ts`/env-var changes and redeploy. No schema change and nothing to undo in the database; reverting returns the deployment to its current broken-sign-in state.
