# Hootden

Hootden — a cozy personal workspace. Keep your own den for solo notes and planning, or open a shared nest to collaborate with others.

Go API (`server/`) + Next.js frontend (`web/`) + Postgres. Sign-in is Google OAuth only; sessions are opaque server-side tokens delivered as an `HttpOnly` cookie.

## Local development

Prerequisites: Go 1.26+, Node 24+, Docker.

1. Start Postgres (the bare `docker compose up` only starts this service — `server`/`web` are gated behind the `full` profile below, for `go run`/`npm run dev` instead):

   ```bash
   docker compose up -d
   ```

2. Copy the env files and fill in Google OAuth credentials:

   ```bash
   cp server/.env.example server/.env
   cp web/.env.example web/.env.local
   ```

   Create an OAuth client in [Google Cloud Console](https://console.cloud.google.com/apis/credentials) with authorized redirect URI `http://localhost:8080/auth/google/callback`, and set `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` in `server/.env`. Leave `COOKIE_DOMAIN` empty for localhost — see the comment in `server/.env.example`.

3. Run the API (applies migrations on startup). Go doesn't load `.env` files itself, so export it first:

   ```bash
   cd server && set -a && source .env && set +a && go run .
   ```

4. Run the frontend (Next.js loads `.env.local` automatically):

   ```bash
   cd web && npm install && npm run dev
   ```

5. Open `http://localhost:3000` and sign in.

### Running the full stack in Docker instead

`docker compose --profile full up --build` builds and runs `postgres` + `server` + `web` together — no local Go/Node toolchain needed. Reads the same `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`GOOGLE_REDIRECT_URL`/`COOKIE_DOMAIN`/`APP_ORIGIN`/`NEXT_PUBLIC_API_ORIGIN` from the shell environment (or a `.env` file at the repo root) rather than the per-service `.env` files above.

### Environment variables

`server/.env.example` and `web/.env.example` are the source of truth; summary:

| Variable | Where | Notes |
|---|---|---|
| `PORT` | server | HTTP port, defaults to `8080` |
| `DATABASE_URL` | server | Postgres connection string |
| `APP_ORIGIN` | server | Exact frontend origin, echoed back as `Access-Control-Allow-Origin` |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | server | From the Google Cloud OAuth client |
| `GOOGLE_REDIRECT_URL` | server | Must match the client's authorized redirect URI exactly |
| `COOKIE_DOMAIN` | server | Empty for a host-only cookie (localhost); `.<domain>` in production so the apex and `api.` subdomain share it |
| `NEXT_PUBLIC_API_ORIGIN` | web | Baked into the client bundle at **build time** — the API origin as seen by the browser |

### Tests

```bash
cd server && go test ./...
cd web && npm run test:e2e   # needs the full-stack profile running: docker compose --profile full up -d --build
```

## Deploying

The Go and Next.js images (`server/Dockerfile`, `web/Dockerfile`) are identical across every target — only environment variables change. Three ways to run them:

**Railway + Vercel (managed).** Order matters because the OAuth client and cookie both encode the domain:

1. Register a domain and point the apex at Vercel, `api.` at Railway.
2. Create the Google OAuth client with redirect URI `https://api.<domain>/auth/google/callback`.
3. Create a Railway project with Postgres; deploy `server/Dockerfile` to it, bind `api.<domain>`.
4. Create a Vercel project from this repo; Vercel uses its own Next.js build pipeline (not the Dockerfile) — set `NEXT_PUBLIC_API_ORIGIN=https://api.<domain>` as a build-time env var, bind the apex domain.
5. Set `COOKIE_DOMAIN=.<domain>` and `APP_ORIGIN=https://<domain>` on the Railway service.

**Self-hosted Docker Compose (VPS or a personal machine).** Same steps 1–2 above for the domain and OAuth client, then `docker compose --profile full up -d --build` on the host in place of the Railway/Vercel steps — same images, same migrations, same variable contract. What's different from the managed path: you provide your own TLS termination and reverse proxy (Caddy, nginx, or Traefik) in front of ports `3000` and `8080`, and DNS points both the apex and `api.` at that one host instead of two platforms. What's identical: the container images, the migration path, and every environment variable above.

Either path, `COOKIE_DOMAIN` must be the shared parent of both hosts (e.g. `.hootden.example`) — a host-only cookie set on one won't be sent to the other.
