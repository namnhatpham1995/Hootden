## 1. Domain and external setup

Ordered first because the domain is encoded into the OAuth client, the cookie, and the CORS configuration. Doing these later means redoing them.

- [ ] 1.1 Register the domain and confirm DNS resolves for both the apex and an `api.` subdomain record
- [ ] 1.2 Create the Google Cloud OAuth client with the authorized redirect URI set to `https://api.<domain>/auth/google/callback`, and verify the client ID and secret are stored as secrets, not committed
- [ ] 1.3 Create the Railway project with a Postgres instance, and verify a `psql` connection succeeds using the connection string
- [ ] 1.4 Create the Vercel project linked to the repository, and verify a placeholder deploy is reachable at the apex domain over HTTPS
- [x] 1.5 Establish the repo layout (`server/` for Go, `web/` for Next.js) and verify both build with a no-op entry point

## 2. Backend skeleton

- [x] 2.1 Initialise the Go module and an HTTP server using stdlib `net/http` with a `/healthz` route, and verify it returns 200
- [x] 2.2 Add a `pgx/v5` pool configured from environment variables, and verify `/healthz` reports database reachability
- [x] 2.3 Wire `goose` migrations from an embedded filesystem, and verify running migrations creates the goose version table
- [x] 2.4 Write migration 001 creating `users`, `sessions`, `workspaces`, and `pages` per design.md, and verify the self-referencing `pages.parent_id` foreign key cascades by deleting a parent row and confirming descendants are gone
- [x] 2.5 Add CORS middleware returning the exact app origin with `Access-Control-Allow-Credentials: true`, and verify a preflight `OPTIONS` request returns the expected headers and that `*` is never sent
- [x] 2.6 Add a 1 MB `http.MaxBytesReader` limit and a shared JSON error response helper, and verify an oversized body is rejected rather than read into memory

## 3. Authentication

- [x] 3.1 Implement the OAuth start endpoint generating random `state` into a `__Host-` prefixed cookie plus a PKCE challenge, and verify the redirect to Google carries both
- [x] 3.2 Implement the callback verifying `state`, exchanging the code, and resolving the user by Google subject identifier, and verify a first-time subject creates a user row while a repeat subject reuses it
- [x] 3.3 Issue sessions as 32 random bytes stored as a SHA-256 hash, and verify the `Set-Cookie` carries `Domain=.<domain>`, `HttpOnly`, `Secure`, `SameSite=Lax` and that the raw token appears nowhere in the database
- [x] 3.4 Implement session-resolving middleware, with tests covering a valid session, an expired session, and an unknown token
- [x] 3.5 Implement sign-out revoking the session server-side, with a test confirming the same token is rejected after sign-out
- [ ] 3.6 Apply the authentication guard to all non-public routes, and verify unauthenticated requests receive 401 and no workspace or page data (blocked: no protected routes exist yet -- `RequireAuth` is implemented and tested in isolation; applying it happens as workspace/page endpoints are added in groups 4-6)
- [x] 3.7 Verify the callback rejects a missing, mismatched, or expired `state` without creating a user or session

## 4. Workspaces

- [ ] 4.1 Create the personal Den in the same transaction as user creation, and verify a new user has exactly one workspace with `personal = true`
- [ ] 4.2 Add an endpoint returning the signed-in user and their Den, and verify it resolves from the session cookie alone
- [ ] 4.3 Implement a shared ownership check used by every page handler, and verify a request for another account's workspace or page returns 404 rather than 403 so existence is not disclosed

## 5. Page tree API

- [ ] 5.1 Implement the tree endpoint returning every page's id, parent, title, and position without document bodies, and verify the response excludes `doc`
- [ ] 5.2 Implement page creation with optional parent, placeholder title when none is given, and last position among siblings, with a test covering root and nested creation
- [ ] 5.3 Implement rename, and verify the new title is returned by the tree endpoint
- [ ] 5.4 Implement the move operation updating parent and position with sibling reindexing inside one transaction, with a test confirming sibling order persists after reorder and after reparent
- [ ] 5.5 Implement the cycle guard by walking `parent_id` upward, with a test confirming a move under the page's own descendant is refused and the tree is unchanged
- [ ] 5.6 Implement delete relying on the cascade, with a test confirming a subtree is fully removed

## 6. Page document API

- [ ] 6.1 Implement fetching a single page's document, and verify it returns an empty document for a newly created page
- [ ] 6.2 Implement document save with well-formed JSON validation ahead of storage, and verify malformed JSON is rejected without writing
- [ ] 6.3 Update `updated_at` on save and confirm last-write-wins by issuing two saves and verifying the later one is stored intact rather than merged

## 7. Frontend foundation and visual system

- [ ] 7.1 Scaffold the Next.js App Router project, and verify the dev server renders a placeholder route
- [ ] 7.2 Define design tokens for the bark-brown surface, honey accent, dusk-blue secondary, plus spacing and a varied radius scale, and verify them on a style reference route
- [ ] 7.3 Load Fredoka, Nunito Sans, and Lora with real fallback stacks, and verify each renders at its intended role rather than falling back
- [ ] 7.4 Build the bear and owl marks as SVGs composed of primitive shapes, and verify each stays legible at 24 px and in a single flat colour
- [ ] 7.5 Build the API client sending `credentials: 'include'` with shared error handling, and verify a 401 response routes to the signed-out state

## 8. Frontend authentication and shell

- [ ] 8.1 Build the signed-out landing screen, and verify the sign-in control reaches Google's consent page
- [ ] 8.2 Handle the post-callback return by loading the current user and entering the Den, and verify a fresh account lands in an empty workspace with no setup step
- [ ] 8.3 Add the sign-out control, and verify it returns to the signed-out state and that a reload does not restore the session
- [ ] 8.4 Build the app shell with sidebar and content region, and verify no workspace switcher, create, or delete affordance is present

## 9. Page tree UI

- [ ] 9.1 Render the tree from the API, and verify nesting and sibling order match the stored data
- [ ] 9.2 Add page creation at root and as a child, and verify a new page opens for editing immediately
- [ ] 9.3 Add inline rename, and verify both the tree entry and the page heading update without a reload
- [ ] 9.4 Add drag to reorder and reparent, and verify the new position survives a reload
- [ ] 9.5 Add delete with a confirmation stating the descendant count, and verify cancelling removes nothing
- [ ] 9.6 Build the empty state featuring the sleeping bear, and verify it appears for a workspace with no pages and offers to create the first one

## 10. Editor

- [ ] 10.1 Configure TipTap with exactly the node and mark set named in the pages spec, and verify each type survives a save and reload round trip
- [ ] 10.2 Load a stored document into the editor, and verify headings, nested lists, and inline marks are restored as written
- [ ] 10.3 Implement autosave debounced at 800 ms with a saved/saving/failed indicator, and verify a change is persisted within 2 seconds of pausing
- [ ] 10.4 Implement retry with backoff on failed saves, and verify with the API unreachable that content stays in the editor and saves once it returns
- [ ] 10.5 Add the `beforeunload` warning while the document is dirty, and verify it does not fire once saved
- [ ] 10.6 Verify checkbox items retain their checked state across reload

## 11. Deploy and end-to-end verification

- [ ] 11.1 Deploy the Go service to Railway and bind the `api.` subdomain, verifying HTTPS and that migrations ran
- [ ] 11.2 Deploy the frontend to Vercel bound to the apex domain, verifying it reaches the API with credentials
- [ ] 11.3 Verify the full flow in iOS Safari with third-party cookies blocked: sign in, create a page, edit, reload, and confirm the session and content persist
- [ ] 11.4 Audit every route and verify no `GET` endpoint changes state, as the CSRF posture depends on it
- [ ] 11.5 Verify a second Google account gets its own Den and cannot reach the first account's pages by direct id

## 12. Documentation

- [ ] 12.1 Update `README.md` with setup, environment variables, and how to run the Go service and Next.js app locally, and verify a clean clone can reach a running app by following it

