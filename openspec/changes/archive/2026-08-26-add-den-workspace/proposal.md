## Why

Hootden is an empty repository with a name and a README. Nothing runs yet. Before any of the collaborative features the product is ultimately for, there needs to be a working vertical slice: a person can sign in, and their notes are saved and come back.

This change delivers that slice — the **Den**, a single-user personal workspace with a page tree and a rich-text editor. It is deliberately the smallest thing that is genuinely usable day to day, so the stack is proven end to end before multi-user concerns are introduced.

One decision is pulled forward out of proportion to its size: the **session cookie's domain shape**. Split-origin deploys (`*.vercel.app` frontend + `*.up.railway.app` backend) are cross-site, so the session cookie requires `SameSite=None`, which iOS Safari blocks. Discovering this after auth exists means rewriting auth. Getting it right costs a domain registration now.

## What Changes

- **Go backend service** — HTTP API, Postgres persistence, deployed on Railway.
- **Next.js frontend** — deployed on Vercel, served from the apex of a single registrable domain, with the API on its `api.` subdomain.
- **Google OAuth sign-in** — the only sign-in method. No passwords, therefore no password reset, no email verification, and no transactional email of any kind.
- **Server-side sessions** — opaque session token in an `HttpOnly; Secure; SameSite=Lax` cookie scoped to the parent domain so apex and `api.` subdomain share it.
- **Workspaces table with a `personal` boolean** — v0 only ever creates `personal = true` rows (Dens). The column exists now so shared workspaces later are a new row type rather than a schema migration plus a rewrite of every page and permission query.
- **Automatic Den provisioning** — a Den is created on first sign-in. There is no workspace picker, no create-workspace flow, no settings screen.
- **Page tree** — pages nest arbitrarily, are reorderable among siblings, and are renamable and deletable.
- **Rich-text page content** — a TipTap/ProseMirror editor persisting its document as JSON, autosaved. Checkbox lists come from the editor's built-in list support, so a page can serve as a to-do list without any task-specific data model.

### Non-goals

Explicitly out of scope. These are deferred by decision, not oversight, and must not be designed for or partially built:

- Nests (shared workspaces), invites, members, roles, permissions beyond "is this your Den"
- Any real-time transport — no SSE, no WebSockets, no presence
- Structured task rows (assignees, due dates, cross-page task views)
- Mobile-specific layout or a reduced mobile editor — v0 targets desktop browsers
- Public page sharing, and therefore the abuse-handling surface that comes with it
- Search, page templates, trash/restore, notifications
- Storage quotas, rate limiting, account deletion, file/image uploads
- Any OAuth provider other than Google

## Capabilities

### New Capabilities

- `accounts`: Google OAuth sign-in, session issuance and verification, sign-out, and the cookie domain and attribute requirements that make sessions work across the apex and `api.` subdomain.
- `workspaces`: The Den — a personal workspace automatically provisioned per account, and the ownership rule that gates all access to its contents.
- `pages`: The page tree (create, rename, nest, reorder, delete) and page document content (load, edit, autosave, supported block types).

### Modified Capabilities

None. This is the project's first change; `openspec/specs/` is empty.

## Impact

**Created from nothing.** There is no existing code to modify — the repository currently contains only `README.md`, `AGENTS.md`, `CLAUDE.md`, and OpenSpec scaffolding.

- **New backend**: Go service (`net/http`, `database/sql` or `pgx`, `golang.org/x/oauth2`), Postgres schema and migrations.
- **New frontend**: Next.js app, TipTap editor, the visual system (warm-dark "den" palette, Fredoka/Nunito Sans/Lora type scale, bear and owl marks built from basic shapes so they read at 24px).
- **New external dependencies**: a registered domain, a Google Cloud OAuth client, a Railway project (service + Postgres), a Vercel project.
- **Cross-cutting decision**: the domain and cookie shape constrains all future auth work. Changing it later invalidates every issued session and requires reworking the OAuth callback and CORS configuration.
- **No breaking changes** — nothing exists to break.

### Assumptions

Recorded rather than asked, as none materially change v0's scope:

- The exact domain is not yet chosen. The requirement is only that one registrable domain serves the app at its apex and the API at `api.<domain>`.
- Deployment targets Vercel (frontend) and Railway (Go service + Postgres). These are replaceable; the same-registrable-domain constraint is not.
- Sign-up and sign-in are the same action — an unrecognized Google account creates a user and a Den. Signup is open, but with no sharing and no uploads in v0, an unwanted account can do nothing beyond consume a row.
