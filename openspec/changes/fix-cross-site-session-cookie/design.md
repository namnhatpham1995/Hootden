## Context

See `proposal.md — Why` for the failure and the evidence. The relevant constraints:

- `setSessionCookie` in `server/internal/auth/cookies.go` is correct and stays untouched. `Secure; SameSite=Lax` is the CSRF posture the whole API rests on, per `archive/2026-08-26-add-den-workspace/design.md` — "CSRF is covered by `SameSite=Lax` plus restrictive CORS", with the accompanying self-imposed rule that no `GET` endpoint changes state.
- The accounts spec already requires the cookie to work across the app and API origins. That requirement is not being changed; this change makes a deployment that violates it detectable.
- The same repository must run on three targets — Railway + Vercel, a rented VPS, and a personal machine — with only environment variables differing. Any check added here has to hold on all three, and must not break `localhost` development, where the app is on `:3000`, the API on `:8080`, and `COOKIE_DOMAIN` is deliberately empty.
- The server does not currently know its own public hostname. Railway terminates TLS and proxies, so the listening port says nothing about the origin a browser sees.

## Goals / Non-Goals

**Goals:**

- Production sign-in works, using the cookie shape already specified.
- A deployment whose cookie cannot reach the app is impossible to leave running silently.
- The check is exact cookie semantics, not a heuristic that could reject a valid deployment.

**Non-Goals:**

- No change to the session mechanism, the cookie attributes, CORS, or the CSRF posture.
- No validation of anything else in the configuration. This is not a general config linter; it covers the one invariant whose violation is invisible until a user reports it.
- No attempt to make Vercel preview deployments work. They get a fresh hostname per push and are cross-site by construction.

## Decisions

### Register a domain; do not relax the cookie

Three ways out of a cross-site session, and only one survives contact with the constraints:

| | Mechanism | Verdict |
|---|---|---|
| Register a domain, apex to Vercel, `api.` to Railway | Cookie stays `SameSite=Lax`, scoped to the shared parent | **Chosen.** Zero application code. It is the plan already written down, minus the step that was skipped. |
| `SameSite=None; Secure` | Cookie becomes third-party | Rejected. Safari's ITP blocks third-party cookies outright, so sign-in would work on desktop Chrome and fail on iPhone — and mobile is a stated constraint. Also discards the CSRF defence, which would then need double-submit tokens to replace. Already rejected by name in the original design. |
| Proxy the API through Next.js rewrites | Requests become same-origin | Rejected. Already rejected in the original design ("adds a network hop and a second deployment to keep in sync"), and it would put every autosave PATCH through a Vercel function. A permanent architectural cost to avoid a one-time domain registration. |

The domain is not a workaround for a code defect. It is the load-bearing decision the original design identified, and the code was written to assume it.

### The startup check compares hosts, not sites

The tempting check is "are the app and API the same site?", which needs the Public Suffix List to answer correctly — a new dependency and an embedded list that goes stale.

It is not needed. Cookie delivery is decidable from the cookie's own rules, with no knowledge of public suffixes:

- `COOKIE_DOMAIN` empty: the cookie is host-only and reaches only the exact host that set it. So the app's host must **equal** the API's host.
- `COOKIE_DOMAIN` set: the cookie reaches that domain and its subdomains. So both the app's host and the API's host must be **equal to or a subdomain of** `COOKIE_DOMAIN`.

That is exact, not approximate. It accepts every valid deployment and rejects this one:

```
  app host                      api host                     COOKIE_DOMAIN     verdict
  ----------------------------  ---------------------------  ---------------   -------
  hootden.vercel.app            hootden-prod.up.railway.app  (empty)           REFUSE  <- today
  localhost                     localhost                    (empty)           ok      <- local dev
  hootden.example               api.hootden.example          hootden.example   ok      <- target
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

**A new required variable breaks existing deploys on the next restart** → Intended, and it fails at boot with a message naming what to set rather than silently. Both live deploy targets are being reconfigured in this change anyway. `docker-compose.yml` and both `.env.example` files get it with a working default, so local development is unaffected.

**A `COOKIE_DOMAIN` that is itself a public suffix passes the check but is rejected by browsers** → e.g. `COOKIE_DOMAIN=vercel.app` with the app on `hootden.vercel.app` satisfies the subdomain rule, but browsers refuse a cookie domain that is a public suffix. Detecting this is the one case that genuinely needs the PSL. Left uncovered: it requires an operator to deliberately set a platform suffix as the cookie domain, and the README's guidance points the other way. The cheap partial guard — rejecting a `COOKIE_DOMAIN` with fewer than two labels — does not catch it and is not worth the false confidence.

**Everyone is signed out once** → The cookie's host changes, so existing cookies are orphaned. Session rows remain valid but unreachable; accounts, Dens, and pages are untouched. One sign-in, no announcement needed at this stage.

**The check cannot see a reverse proxy that rewrites the host** → On the self-hosted target, Caddy/nginx/Traefik sits in front and `API_ORIGIN` describes the public origin, not the container's. That is the correct thing to configure and what the README will say, but a proxy misconfiguration remains outside what a startup check can observe.

**DNS propagation makes the cutover non-atomic** → During propagation some browsers resolve the old hostnames and continue to fail. Mitigated by binding the custom domains and setting the variables before switching DNS, so the new origins are live the moment records resolve.

## Migration Plan

1. Register the domain; point the apex at Vercel and `api.` at Railway.
2. Bind both custom domains on their platforms and let certificates issue. The platform hostnames keep working throughout — nothing is removed yet.
3. Set `APP_ORIGIN`, `API_ORIGIN`, and `COOKIE_DOMAIN` on Railway, and `NEXT_PUBLIC_API_ORIGIN` on Vercel. Vercel bakes its value in at build time, so this needs a redeploy, not just a variable change.
4. Verify sign-in on the custom domain before announcing anything.
5. Delete the `diagnostic-probe-delete-me@example.invalid` account left by the investigation.

**Rollback:** revert the environment variables and redeploy. The platform hostnames still resolve, and reverting returns the deployment to its current broken-sign-in state — so rollback is only useful for a problem worse than the one being fixed. There is no schema change and nothing to undo in the database.
