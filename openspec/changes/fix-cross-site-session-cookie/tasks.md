## 1. Domain and origin cutover

Ordered first, and deliberately: this group is what actually fixes sign-in, and it sets `API_ORIGIN` before group 2 makes the server require it. Reversing the order would take the API down on its next restart instead of just leaving sign-in broken. No application code changes here — this is the unchecked group 1 of `archive/2026-08-26-add-den-workspace/tasks.md`, finished at last.

- [ ] 1.1 Register the domain and add DNS records for the apex and an `api.` subdomain, and verify both resolve from a machine outside the registrar's network
- [ ] 1.2 Bind the apex as a custom domain on the Vercel project and verify it serves the app over HTTPS with a valid certificate, with `hootden.vercel.app` still reachable
- [ ] 1.3 Bind `api.<domain>` as a custom domain on the Railway service and verify `https://api.<domain>/healthz` returns `{"status":"ok"}` with a valid certificate
- [ ] 1.4 Set `APP_ORIGIN=https://<domain>`, `API_ORIGIN=https://api.<domain>`, and `COOKIE_DOMAIN=<domain>` on Railway, and verify an `OPTIONS` preflight to `https://api.<domain>/auth/login` carrying `Origin: https://<domain>` returns that exact origin in `Access-Control-Allow-Origin` alongside `Access-Control-Allow-Credentials: true`
- [ ] 1.5 Set `NEXT_PUBLIC_API_ORIGIN=https://api.<domain>` on Vercel and trigger a redeploy — it is baked in at build time, so a variable change alone does nothing — then verify the deployed bundle requests `api.<domain>` and not the Railway hostname
- [ ] 1.6 Register an account through the UI on the apex domain and verify the response's `Set-Cookie` carries `Domain=<domain>`, that the browser stores it rather than flagging it in DevTools' Network → Cookies tab, that `GET /me` returns 200, and that the Den renders instead of the landing screen
- [ ] 1.7 Delete the `diagnostic-probe-delete-me@example.invalid` account left behind by the investigation (`DELETE FROM users WHERE email = 'diagnostic-probe-delete-me@example.invalid'` — the Den, pages, and sessions cascade) and verify no row remains

## 2. Startup configuration check

Shippable on its own once group 1 has set the variables. This is the whole of the spec delta.

- [ ] 2.1 Add `API_ORIGIN` to `config.Config` and `config.Load()` as a required variable alongside `DATABASE_URL`, and verify `Load()` returns an error naming it when unset
- [ ] 2.2 Add the cookie-reachability check to `config.Load()` per design.md's host comparison — with `COOKIE_DOMAIN` empty the app and API hosts must be equal, and with it set both must equal or be subdomains of it — returning an error that names the conflicting values, and verify with table tests covering each row of design.md's table, including that `localhost`/`localhost`/empty passes and that today's `hootden.vercel.app` + `up.railway.app` pair is refused
- [ ] 2.3 Verify the server exits at startup rather than serving, by booting it with the failing combination and confirming it never binds the port and logs a message naming `APP_ORIGIN`, `API_ORIGIN`, and `COOKIE_DOMAIN`
- [ ] 2.4 Add `API_ORIGIN` to `server/.env.example` (defaulting to `http://localhost:8080`) and to `docker-compose.yml`'s `server` service, and verify both `go run .` from `server/.env` and `docker compose --profile full up` still start and pass the existing e2e suite

## 3. Verification across deploy targets

Broader than any single task above: this is the group-13 verification the original change never ran.

- [ ] 3.1 Verify the full signed-in flow on the apex domain in desktop Chrome — register, create a page, edit, reload, sign out, sign back in — and confirm content persists across the sign-out
- [ ] 3.2 Verify the same flow on iOS Safari with "Prevent Cross-Site Tracking" enabled, confirming the session survives, since this is the browser the original design named as the reason for choosing `SameSite=Lax` and it has never been tested
- [ ] 3.3 Audit every route in `main.go` and verify no `GET` endpoint changes state, since the CSRF posture this change preserves depends on it (the outstanding task 13.4 from the archived change)
- [ ] 3.4 Run the full-stack Docker Compose profile on a second host with `API_ORIGIN` and `APP_ORIGIN` pointing at that host, and verify sign-in works there too, confirming the check holds on the self-hosted target and not just the managed one

## 4. Documentation

- [ ] 4.1 Add `API_ORIGIN` to the README's environment-variable table and to the deploy section's variable list, and verify the table matches what `config.Load()` actually requires
- [ ] 4.2 Add an explicit note to the README's deploy section that platform-issued hostnames (`*.vercel.app` with `*.up.railway.app`) cannot deliver the session, that no value of `COOKIE_DOMAIN` rescues them because a public suffix is not a legal cookie domain, and that Vercel preview deployments are cross-site by construction — and verify someone following the deploy steps encounters this before creating the projects, not after
- [ ] 4.3 Re-verify the README's local-development steps end to end on a clean checkout with the new required variable in place, following them as someone unfamiliar with the project would
