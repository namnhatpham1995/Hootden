# Hootden

Hootden — a cozy personal workspace. Keep your own den for solo notes and planning, or open a shared nest to collaborate with others.

Go API (`server/`) + Next.js frontend (`web/`) + Postgres. Sign in with email and password, or with Google OAuth if it's configured; sessions are opaque server-side tokens delivered as an `HttpOnly` cookie.

## Local development

Prerequisites: Go 1.26+, Node 24+, Docker.

1. Start Postgres (the bare `docker compose up` only starts this service — `server`/`web` are gated behind the `full` profile below, for `go run`/`npm run dev` instead):

   ```bash
   docker compose up -d
   ```

2. Copy the env files:

   ```bash
   cp server/.env.example server/.env
   cp web/.env.example web/.env.local
   ```

   Leave `COOKIE_DOMAIN` empty for localhost — see the comment in `server/.env.example`. That's enough to sign in with email and password; the `GOOGLE_*` variables are optional. To also offer Google sign-in, create an OAuth client in [Google Cloud Console](https://console.cloud.google.com/apis/credentials) with authorized redirect URI `http://localhost:8080/auth/google/callback` and set `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` in `server/.env` — the frontend hides the Google button on its own when they're unset.

3. Run the API (applies migrations on startup). Go doesn't load `.env` files itself, so export it first:

   ```bash
   cd server && set -a && source .env && set +a && go run .
   ```

4. Run the frontend (Next.js loads `.env.local` automatically):

   ```bash
   cd web && npm install && npm run dev
   ```

5. Open `http://localhost:3000` and create an account.

### Running the full stack in Docker instead

`docker compose --profile full up --build` builds and runs `postgres` + `server` + `web` together — no local Go/Node toolchain needed. Reads the same `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`GOOGLE_REDIRECT_URL`/`COOKIE_DOMAIN`/`APP_ORIGIN`/`API_ORIGIN`/`API_PROXY_TARGET` from the shell environment (or a `.env` file at the repo root) rather than the per-service `.env` files above.

### Environment variables

`server/.env.example` and `web/.env.example` are the source of truth; summary:

| Variable | Where | Notes |
|---|---|---|
| `PORT` | server | HTTP port, defaults to `8080` |
| `DATABASE_URL` | server | Postgres connection string |
| `APP_ORIGIN` | server | Exact frontend origin, echoed back as `Access-Control-Allow-Origin` |
| `API_ORIGIN` | server | This API's own public origin, as the browser sees it. Required — the server refuses to start if it and `APP_ORIGIN`/`COOKIE_DOMAIN` describe a session cookie that could never reach the app; see [Deploying](#deploying). On the managed Railway + Vercel target this is set equal to `APP_ORIGIN` (the Vercel origin), since the browser only ever talks to Vercel |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | server | Optional — from the Google Cloud OAuth client. Leave unset and only email/password sign-in is offered |
| `GOOGLE_REDIRECT_URL` | server | Optional, must match the client's authorized redirect URI exactly when Google is configured |
| `COOKIE_DOMAIN` | server | Empty for a host-only cookie (localhost, or the managed Railway + Vercel target); `<domain>` self-hosted (no leading dot) so the apex and `api.` subdomain share it |
| `API_PROXY_TARGET` | web | Baked into the routes manifest at **build time** — the origin `/api/*` requests are rewritten to server-side (the Railway origin on the managed target). Defaults to `http://localhost:8080` for local dev |

### Tests

```bash
cd server && go test ./...
cd web && npm run test:e2e   # needs the full-stack profile running: docker compose --profile full up -d --build
```

## Rate limiting

`POST /auth/login` and `POST /auth/register` are each limited to 20 attempts per 15 minutes, counted separately per submitted email and per client address (both must be under the limit), and separately again between login and register so hammering one doesn't spend the other's budget. A refused attempt gets `429` with `Retry-After` before any password is checked. The counters live in the server process only — they reset to zero on every restart, so if you're testing sign-in locally and hit a `429` after restarting the server a few times in a row, that's the counter resetting cleanly, not a bug.

## Deploying

The Go and Next.js images (`server/Dockerfile`, `web/Dockerfile`) are identical across every target — only environment variables change. Two ways to run them:

**Railway + Vercel (managed) needs no custom domain.** A bare Vercel deployment (`<project>.vercel.app`) and a bare Railway deployment (`<project>.up.railway.app`) are on different registrable domains — normally that makes every request between them cross-site, and the session cookie (`Secure; SameSite=Lax` by design, not something to relax) would be refused regardless of `COOKIE_DOMAIN`, since a public suffix is not a legal cookie domain. The fix isn't a domain, it's a proxy: Next.js `rewrites()` forwards every `/api/*` request server-side to Railway (`API_PROXY_TARGET`), so the browser only ever talks to the Vercel origin and the session cookie is same-origin. This applies to Vercel preview deployments too — each gets a fresh `*.vercel.app` URL per push, but `API_PROXY_TARGET` covers the `preview` environment as well as production, so the same proxy mechanism applies there.

1. Optional: create a Google OAuth client with redirect URI `https://<vercel-origin>/api/auth/google/callback` — against the Vercel origin (through the proxy), not Railway's bare origin, since Google redirects the browser there directly and the response's `Set-Cookie` only lands on the origin the browser actually requested. Skip this and email/password is the only sign-in method.
2. Create a Railway project with Postgres; deploy `server/Dockerfile` to it.
3. Create a Vercel project from this repo; Vercel uses its own Next.js build pipeline (not the Dockerfile) — set `API_PROXY_TARGET=https://<railway-origin>` as a build-time env var. It's baked into the routes manifest at build time, so changing it later needs a redeploy, not just a variable edit.
4. Set `APP_ORIGIN` and `API_ORIGIN` on the Railway service both to the Vercel origin (e.g. `https://<project>.vercel.app`), leave `COOKIE_DOMAIN` empty, plus the `GOOGLE_*` variables if you created a client in step 1. If `APP_ORIGIN`/`API_ORIGIN`/`COOKIE_DOMAIN` describe a session cookie that couldn't reach the app, the server refuses to start rather than deploying broken — see the `API_ORIGIN` row above.

**Self-hosted Docker Compose (VPS or a personal machine)** isn't behind the Vercel proxy above, so it keeps the original shape and still needs a real domain: register one, point the apex at the host running `web` and `api.` at the host running `server` (the same host for both is fine), optionally create a Google OAuth client with redirect URI `https://api.<domain>/auth/google/callback`, then `docker compose --profile full up -d --build` — same images, same migrations, same variable contract as the managed path, except `APP_ORIGIN=https://<domain>`, `API_ORIGIN=https://api.<domain>`, and `COOKIE_DOMAIN=<domain>` (no leading dot, the shared parent so a host-only cookie set on one subdomain still reaches the other). What's different from the managed path: you provide your own TLS termination and reverse proxy (Caddy, nginx, or Traefik) in front of ports `3000` and `8080`, and DNS points both hostnames at that one host instead of two platforms. What's identical: the container images, the migration path, and every environment variable above.
