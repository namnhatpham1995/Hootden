## Context

Greenfield. The repository contains documentation and OpenSpec scaffolding only — there is no code, no schema, and no deployed environment. Every decision here is a first decision, which means none of them are constrained by existing structure and all of them set precedent.

See `proposal.md` — Why for motivation. Two constraints shape the approach:

- **The product this becomes is multi-user.** Shared workspaces, real-time sync, and a mobile surface are all planned and all excluded from this version. The design must avoid choices that would have to be undone, without building any of them.
- **The frontend and backend are deployed to different platforms.** Next.js on Vercel, Go on Railway. They are therefore different origins, which makes session delivery the one genuinely load-bearing decision in this change.
- **Railway and Vercel are the first deploy target, not the only one.** The same repository must also run as a self-hosted Docker Compose stack, on a rented VPS or on a personal machine, without a second codebase or a fork. Every deploy-facing decision (the domain/cookie shape, the container images) is made to hold across all three targets, not just the managed one.

## Goals / Non-Goals

**Goals:**

- A working vertical slice: sign in, write, reload, still there.
- A session mechanism that works in every mainstream browser, including iOS Safari with third-party cookies blocked, and that does not need rework when shared workspaces arrive.
- A data model where adding Nests later is inserting rows, not migrating tables and rewriting queries.
- Minimum dependency surface. The stack should be legible to one person returning to it after two months away.

**Non-Goals** (design-level, beyond the proposal's scope exclusions):

- No abstraction layers anticipating features in the non-goals list. No `Workspace` interface with one implementation, no permission framework for a single permission, no sync layer with one transport.
- No marketing site. The signed-out state is a single screen with a sign-in button; a real landing page comes with the public push.
- No mobile layout work. Responsive enough not to be broken, not designed for phones.
- No *unit* test infrastructure beyond what the logic warrants — the tree operations and the session lifecycle get tests; CRUD passthrough does not. This does not extend to end-to-end coverage, see the Playwright decision below.

## Decisions

### One registrable domain, apex and `api.` subdomain

The web app is served from `hootden.example` (Vercel), the API from `api.hootden.example` (Railway).

The session cookie is set with `Domain=.hootden.example; HttpOnly; Secure; SameSite=Lax; Path=/`. Because both hosts share a registrable domain, requests from the app to the API are **same-site**, and the cookie is delivered without any third-party cookie permission.

**Alternative rejected — platform-provided hostnames** (`hootden.vercel.app` + `hootden.up.railway.app`). These are different registrable domains, so requests between them are cross-site. The cookie would need `SameSite=None; Secure`, making it a third-party cookie. Safari's Intelligent Tracking Prevention blocks those outright; Chrome is restricting them. Authentication would work in local development and on desktop Chrome and fail on iPhone — discovered from a user, after launch. The domain registration is the cheapest bug prevention available in this change, and it must happen before the OAuth client is configured.

**Same-site is not same-origin.** The two hosts differ, so browser requests from app to API are cross-origin and CORS still applies. The API must return `Access-Control-Allow-Origin` set to the exact app origin (never `*`, which is incompatible with credentials) and `Access-Control-Allow-Credentials: true`. The frontend must send `credentials: 'include'`. These are separate mechanisms solving separate problems and both are required.

### Opaque server-side sessions, not JWTs

Sign-in generates 32 bytes from a cryptographically secure source, base64url-encoded, and returns it as the cookie value. The server stores only its SHA-256 hash alongside `user_id` and `expires_at`. Verification hashes the presented token and looks it up.

**Why not a JWT:** the spec requires sign-out to revoke the session server-side. A JWT cannot be revoked without a server-side denylist — which is a session table with extra steps and a signature library. Storing the hash rather than the token means a database leak does not hand an attacker live sessions.

### CSRF is covered by `SameSite=Lax` plus restrictive CORS

`SameSite=Lax` withholds the cookie on cross-site `POST`/`PATCH`/`DELETE`, and cross-origin scripted requests need CORS approval that only the app's own origin receives. That combination is sufficient here, with one constraint it imposes:

**No state-changing `GET` endpoints.** `Lax` permits the cookie on top-level cross-site `GET` navigation, so any mutation reachable by `GET` would be exposed. All mutations use `POST`, `PATCH`, or `DELETE`.

Accepted residual risk: the cookie is sent to every subdomain of the registrable domain. Nothing untrusted may be hosted on one. If that changes, add double-submit CSRF tokens.

### Google OAuth with PKCE; callback lands on the API

The API owns the OAuth client secret and performs the code exchange, then redirects to the app. The `state` value is random and stored in a short-lived `__Host-` prefixed cookie on the API origin — `__Host-` forbids a `Domain` attribute, correctly confining it to the single host that sets and reads it. PKCE is used even though this is a confidential client; it costs one parameter.

Accounts are keyed on the provider's stable subject identifier, not the email address. Emails at Google can change; the subject does not. The email is stored for display only.

### Go standard library `net/http`, no router or framework

Go 1.22+ `ServeMux` supports method and wildcard patterns (`GET /pages/{id}`), which is the entirety of what a router was needed for. Middleware is plain `func(http.Handler) http.Handler`.

Dependencies are limited to `jackc/pgx/v5` (Postgres driver — better `JSONB` handling than `database/sql`), `golang.org/x/oauth2`, and `pressly/goose` for migrations. Migrations are the one place a hand-rolled forty lines is a bad trade: a subtly wrong migration runner corrupts state in a way that is expensive to unpick.

### Data model

```
users (id, google_sub UNIQUE, email, created_at)
sessions (token_hash PK, user_id → users, expires_at, created_at)
workspaces (id, owner_id → users, personal BOOL, created_at)
pages (id, workspace_id → workspaces, parent_id → pages NULL,
       title, doc JSONB, position INT, created_at, updated_at)
```

**`workspaces.personal` exists although only `true` is ever written.** One column now versus, later, a migration plus rewriting every page query and every ownership check to handle two workspace kinds. This is the rare case where anticipating a future feature is cheaper than deferring it — and it is a column, not an abstraction.

**`pages.parent_id` is self-referencing with `ON DELETE CASCADE`.** Postgres cascades recursively, so deleting a subtree is one `DELETE` statement rather than an application-side tree walk.

### Integer sibling positions, reindexed on move

`position` is an integer, unique per `(workspace_id, parent_id)`. Moving a page renumbers its siblings inside a transaction.

**Alternative rejected — fractional indexing.** Fractional keys make a move a single-row update with no renumbering, and are the correct answer when siblings number in the thousands or when concurrent reorders must merge. Neither applies: this is one user, one tab authoritative, and a realistic sibling count under fifty. It would add a dependency and a class of key-exhaustion edge cases to solve a problem that does not exist yet.

> `ponytail:` integer positions with sibling reindex on move. Switch to fractional indexing if sibling counts grow large or concurrent reordering becomes real (it cannot until shared workspaces exist).

### The whole tree is loaded at once; no recursive CTE

`SELECT id, parent_id, title, position FROM pages WHERE workspace_id = $1` returns every node without document bodies. The tree is assembled in memory. At a realistic personal-workspace size this is a few kilobytes.

This falls out well: the descendant count for the delete confirmation is a client-side count, and the cyclic-move check is walking `parent_id` upward in a loop rather than a recursive CTE. Documents are fetched per page, never with the tree.

### Documents stored as ProseMirror JSON in `JSONB`

TipTap is ProseMirror; its native document format is JSON, and it goes into `JSONB` unchanged.

**Alternative rejected — Markdown.** Round-tripping loses fidelity precisely where this product needs it: nested checkbox lists, mixed inline marks, and arbitrary block nesting. Every serialization boundary is a place for a checkbox to lose its state.

The request body carries a hard size limit (1 MB) via `http.MaxBytesReader`, and the document is validated as well-formed JSON before it is stored. This is a trust boundary; it gets checked regardless of how small the code would otherwise be.

### Autosave debounces at 800 ms

The spec requires persistence within 2 seconds of a pause. Debouncing *at* 2 seconds would mean the write starts at 2 seconds and lands later, violating it. 800 ms leaves room for the round trip.

Failed saves keep the dirty document in the editor and retry with backoff; the editor is never cleared on a failed response. A `beforeunload` handler warns while the document is dirty.

### Next.js App Router; the app is client-rendered and calls Go directly

The signed-out screen is server-rendered. Everything behind auth is client-rendered and talks to `api.hootden.example` with `credentials: 'include'`.

**Alternative rejected — proxying the API through Next.js route handlers.** It adds a network hop and a second deployment to keep in sync, and the usual reason to do it (hiding a cross-site cookie problem) does not exist once the domain decision above is made.

### Visual system

The project's anti-slop rules apply directly here, and "cute" is where generic defaults are strongest. The deliberate departures:

- **Light and dark are both first-class, light is the default.** Every colour is a token pair (e.g. `--surface-light` / `--surface-dark`), not a dark palette with a light palette bolted on later — building both together is the same token work as building one plus roughly a third more, versus a second pass through every component if dark were added after the fact. Base surface is warm paper in light mode, deep bark brown in dark — neither slate nor near-black in either mode, and neither the generic cute-app pastel nor a generic developer-tool dark. One dominant colour, one real accent (honey/amber), with a dusk blue used sparingly for owl-side accents, in both modes. No gradient headings, no glow shadows.
- **Three-step type scale, not one family.** Display in Fredoka (carries the chibi register), UI in Nunito Sans (rounded, legible, quiet), document body in Lora — a reading app earns a reading serif, and it is the choice competitors do not make.
- **Mascots built from six to eight primitive shapes each.** Bear and owl are constructed geometrically rather than illustrated or generated. This guarantees they read at 24 px and in one colour, exports cleanly to SVG, and avoids the tells of generated artwork. Built as outline shapes with a fill token, not a baked-in colour, so they render correctly in both themes without a second asset.
- **Mascot placement is restricted** to empty states, loading, avatar fallback, and the favicon. Not feature icons, not on cards. An empty workspace showing a sleeping bear turns "nothing here" into "not yet"; a mascot on every surface becomes wallpaper.

### Docker images for every deploy target, Vercel's own pipeline for its own target

`server/Dockerfile` (multi-stage Go build, small final image) and `web/Dockerfile` (Next.js with `output: 'standalone'` in `next.config.ts`, so the image doesn't carry the full `node_modules` tree) let the same repository run as a `docker compose` stack on a VPS or a personal machine. `docker-compose.yml` gains a full-stack profile alongside the existing dev-only Postgres service.

Vercel does not build from a Dockerfile — it uses its own Next.js build pipeline regardless of one being present, so adding one costs the managed path nothing. Railway can build a Go service from a Dockerfile directly, so the same image serves both Railway and self-hosted Docker without divergence. No target requires its own fork of the build.

**Alternative rejected — a separate deployment repo or config per target.** Would drift the moment either copy changes. One set of images, three ways to run them (Vercel's pipeline for Next.js; Railway building the Go Dockerfile; `docker compose` for everything self-hosted) keeps the deployable artifact singular.

### Playwright E2E for the flows a unit test can't see

Unit and integration tests (already in place for the backend) verify individual functions and handlers in isolation. They cannot catch a wiring failure across the frontend/backend boundary — a cookie that never gets sent, a form that never calls the endpoint it's bound to. Playwright covers exactly the flows named as safety-critical: sign-in, page CRUD, drag-reorder, and autosave.

**Google's real consent screen cannot be driven by an automated browser** — bot detection, real credentials, and potentially 2FA make it unreliable even as a flake-prone test, not just a security concern. Authenticated E2E tests seed a session directly: insert a user and Den row and a valid session row into Postgres, set the resulting session cookie, and start the browser already signed in. This exercises the same `RequireAuth` code path a real session would, without a bypass endpoint that could ever be reachable outside tests.

**Alternative rejected — a test-only auth bypass endpoint gated by an environment flag.** One misconfigured environment variable in production turns a testing convenience into an authentication bypass. Seeding the database directly has no such failure mode: there is no code path in the shipped binary that skips authentication, tests or otherwise.

This pairs naturally with the Docker work above: the full-stack `docker compose` profile that self-hosting needs is the same stack Playwright runs its suite against in CI.

## Risks / Trade-offs

**The domain is not registered yet, and auth cannot be built correctly without it** → It is the first task. The Google OAuth client's authorized redirect URI must be the final `api.` host; configuring it against a temporary hostname means redoing it and re-testing the cookie path.

**Hard delete with no trash means real data loss** → Mitigated by a confirmation stating the descendant count. Accepted for this version; trash/restore is queued for the next. The `ON DELETE CASCADE` decision does not obstruct adding soft delete later — it becomes a `deleted_at` filter with cascade retained for permanent purge.

**Editor schema drift can orphan stored documents** → If the set of supported node types is narrowed or renamed later, existing documents may contain nodes the new schema rejects, and ProseMirror discards unknown content on parse. Mitigation: the supported node set in the spec is a contract; changing it requires a migration over stored documents, not just an editor config edit.

**Google-only sign-in means a lost Google account is a lost Hootden account** → Accepted. There is no recovery path and no second factor to fall back on because there is no password. Adding a second provider linked by verified email addresses this later without changing the session design.

**Single instance on whichever target is running, sessions and data are in Postgres** → Restarts, redeploys, and switching between Railway/VPS/personal-machine are all safe; nothing session-related lives in process memory, so no deploy target is special-cased. This is only true because there is no real-time transport in this version — the moment SSE arrives, in-process subscriber state exists and multi-instance becomes a real question, regardless of which target it's running on.

**Three deploy targets means three places a misconfiguration can hide** → Mitigated by keeping the container images identical across targets (see the Docker decision above) — only environment variables differ, never the build. The domain/cookie shape is also target-agnostic by design (session cookie's `Domain` covers the apex + `api.` subdomain regardless of what's hosting either).

**Self-imposed constraint: no state-changing `GET`** → Easy to violate accidentally, and the failure is silent (a working endpoint with a CSRF hole). Worth a review check rather than a framework.

## Migration Plan

Greenfield; there is no existing system, no data to migrate, and no rollback target. Deployment is ordered by dependency, for the initial (Railway + Vercel) target:

1. Register the domain. Everything downstream encodes it.
2. Create the Google Cloud OAuth client with the final `api.` redirect URI.
3. Provision Railway Postgres; run migrations via goose.
4. Deploy the Go service to Railway (building `server/Dockerfile`); bind `api.` to it.
5. Deploy Next.js to Vercel (its own build pipeline, not the Dockerfile); bind the apex to it.
6. Verify the session end to end in iOS Safari with third-party cookies blocked, before anything else is built on top of auth.

Self-hosting on a VPS or personal machine follows the same domain/OAuth-client setup (steps 1-2), then substitutes `docker compose up` against the full-stack profile for steps 3-5 — same images, same migrations, same environment-variable contract, just one host instead of two managed platforms.

Migrations are forward-only in practice; goose `down` exists as a development escape hatch, not a production rollback strategy.

## Open Questions

Deferrable without changing the specs, the approach, or the task breakdown:

- The exact domain name. It is configuration; the design depends only on it being one registrable domain with an `api.` subdomain.
- Railway and Vercel hosting regions.
- Whether the second identity provider (likely GitHub) arrives with the public push or later. The account model already keys on provider subject, so it changes nothing here.
- The reverse proxy / TLS termination for self-hosted Docker Compose (VPS or personal machine) — Caddy, nginx, or Traefik. Doesn't affect the images or the application code either way, only the deploy-time compose/proxy config for that target.
