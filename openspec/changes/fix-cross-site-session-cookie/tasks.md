## 1. Reverse-proxy cutover (replaces domain registration)

Ordered first, and deliberately: this group is what actually fixes sign-in. It replaces the original group 1 of this change, which planned to register a domain — the deployment owner has ruled that out for what is a portfolio project (see `proposal.md`/`design.md`). No custom domain, DNS, or certificate work; the fix is entirely in `web/` plus two deployment variables.

- [x] 1.1 Add `async rewrites()` to `web/next.config.ts` forwarding `/api/:path*` to `${API_PROXY_TARGET}/:path*`, defaulting `API_PROXY_TARGET` to `http://localhost:8080` for local dev, and verify `next build` produces a routes manifest containing the rule
- [x] 1.2 Change `web/lib/api.ts` to call relative `/api/...` paths instead of building an absolute URL from `API_ORIGIN`/`NEXT_PUBLIC_API_ORIGIN`, and update its doc comment that currently cites the now-reversed "never through a Next.js route handler" decision
- [x] 1.3 Rename `NEXT_PUBLIC_API_ORIGIN` to `API_PROXY_TARGET` in `web/Dockerfile` (still a build `ARG`, now feeding the rewrite manifest instead of the client bundle), `web/.env.example`, and `docker-compose.yml`'s `web` service build args
- [ ] 1.4 Set `API_PROXY_TARGET=<railway-origin>` on Vercel and trigger a redeploy — it is baked into the routes manifest at build time, so a variable change alone does nothing — then verify the deployed app's `/api/*` requests reach Railway, and check whether a Vercel preview deployment's `/api/*` calls now also reach it (expected side effect of the proxy; note the actual result for task 5.3)
- [ ] 1.5 Set `API_ORIGIN` on Railway equal to `APP_ORIGIN`'s value (the Vercel origin) and leave `COOKIE_DOMAIN` empty, and verify the server still starts — the existing `checkCookieReachability` check from group 2 passes because the hosts are now equal
- [ ] 1.6 Register an account through the UI on the Vercel origin and verify the response's `Set-Cookie` carries no `Domain` attribute, that the browser stores it rather than flagging it in DevTools' Network → Cookies tab, that `GET /api/me` returns 200 through the proxy, and that the Den renders instead of the landing screen
- [ ] 1.7 Delete the `diagnostic-probe-delete-me@example.invalid` account left behind by the investigation (`DELETE FROM users WHERE email = 'diagnostic-probe-delete-me@example.invalid'` — the Den, pages, and sessions cascade) and verify no row remains

## 2. Startup configuration check

Shippable on its own once group 1 has set the variables. This is the whole of the spec delta. Already implemented and merged — unaffected by the group 1 rewrite above: the check compares hosts, not sites, and `APP_ORIGIN == API_ORIGIN` with `COOKIE_DOMAIN` empty (the shape group 1 now deploys) already satisfies it. No code changes needed for the proxy pivot.

- [x] 2.1 Add `API_ORIGIN` to `config.Config` and `config.Load()` as a required variable alongside `DATABASE_URL`, and verify `Load()` returns an error naming it when unset
- [x] 2.2 Add the cookie-reachability check to `config.Load()` per design.md's host comparison — with `COOKIE_DOMAIN` empty the app and API hosts must be equal, and with it set both must equal or be subdomains of it — returning an error that names the conflicting values, and verify with table tests covering each row of design.md's table, including that `localhost`/`localhost`/empty passes and that today's `hootden.vercel.app` + `up.railway.app` pair is refused
- [x] 2.3 Verify the server exits at startup rather than serving, by booting it with the failing combination and confirming it never binds the port and logs a message naming `APP_ORIGIN`, `API_ORIGIN`, and `COOKIE_DOMAIN`
- [x] 2.4 Add `API_ORIGIN` to `server/.env.example` (defaulting to `http://localhost:8080`) and to `docker-compose.yml`'s `server` service, and verify both `go run .` from `server/.env` and `docker compose --profile full up` still start and pass the existing e2e suite

## 3. Verification across deploy targets

Broader than any single task above: this is the group-13 verification the original change never ran.

- [ ] 3.1 Verify the full signed-in flow on the Vercel origin in desktop Chrome — register, create a page, edit, reload, sign out, sign back in — and confirm content persists across the sign-out
- [ ] 3.2 Verify the same flow on iOS Safari with "Prevent Cross-Site Tracking" enabled, confirming the session survives, since this is the browser the original design named as the reason for choosing `SameSite=Lax` and it has never been tested
- [ ] 3.3 Audit every route in `main.go` and verify no `GET` endpoint changes state, since the CSRF posture this change preserves depends on it (the outstanding task 13.4 from the archived change)
- [ ] 3.4 Run the full-stack Docker Compose profile on a second host with a real domain, `API_ORIGIN`/`APP_ORIGIN` pointing at it, and `COOKIE_DOMAIN` set, and verify sign-in works there too — this target is untouched by the group 1 pivot, so it still exercises the domain-based branch of `checkCookieReachability`

## 4. Documentation

Already implemented and merged. Written for the domain plan, since that was still the direction at the time — group 5 below re-revises the parts that now describe a plan this change no longer follows. Kept checked as an accurate record of what was done, not reopened.

- [x] 4.1 Add `API_ORIGIN` to the README's environment-variable table and to the deploy section's variable list, and verify the table matches what `config.Load()` actually requires — also fixed a stale leading-dot `COOKIE_DOMAIN` example in `config.go` and `server/.env.example` that predated this change and would have made the new startup check refuse a correctly configured deployment
- [x] 4.2 Add an explicit note to the README's deploy section that platform-issued hostnames (`*.vercel.app` with `*.up.railway.app`) cannot deliver the session, that no value of `COOKIE_DOMAIN` rescues them because a public suffix is not a legal cookie domain, and that Vercel preview deployments are cross-site by construction — and verify someone following the deploy steps encounters this before creating the projects, not after
- [x] 4.3 Re-verify the README's local-development steps end to end on a clean checkout with the new required variable in place, following them as someone unfamiliar with the project would — cloned fresh, followed steps 1-5 literally (`docker compose up -d`, copied both `.env.example` files unmodified, ran the API and the frontend, registered an account through the UI), Den rendered with no manual fixes needed

## 5. Documentation for the proxy path

Group 4 documented the domain plan; this re-revises the parts of it that now describe a plan this change no longer follows.

- [ ] 5.1 Rewrite the README's "Deploying" → "Railway + Vercel" steps to describe the proxy cutover instead of domain registration: no DNS, no custom domain binding, `API_PROXY_TARGET` set on Vercel, `API_ORIGIN` set equal to `APP_ORIGIN` on Railway, `COOKIE_DOMAIN` left empty
- [ ] 5.2 Update the README's environment-variable table: rename `NEXT_PUBLIC_API_ORIGIN` to `API_PROXY_TARGET` with its new description, and note that `API_ORIGIN` on the Railway (managed) target is set equal to `APP_ORIGIN`
- [ ] 5.3 Replace the README's "platform-issued hostnames cannot work" warning with an explanation of why the proxy makes them work, using task 1.4's actual result for the Vercel-preview caveat — keep it only if previews were confirmed still broken, drop it if they turned out to work too
- [ ] 5.4 Re-verify the updated deploy steps by actually performing them against the live Vercel/Railway deployment, confirming sign-in works end to end on the platform-issued hostnames with no custom domain
