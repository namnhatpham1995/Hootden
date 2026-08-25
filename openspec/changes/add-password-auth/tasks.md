## 1. Data model

- [ ] 1.1 Write migration `0002_password_auth.sql`: add `password_hash TEXT NULL` to `users`, drop `NOT NULL` on `google_sub`, add `CHECK (google_sub IS NOT NULL OR password_hash IS NOT NULL)`, add a `UNIQUE` index on `email`, and verify a password-only row (`google_sub` null) inserts cleanly while a row with neither set is rejected by the check constraint
- [ ] 1.2 Add the `golang.org/x/crypto/bcrypt` dependency, and verify `go build ./...` succeeds with it in `go.mod`/`go.sum`

## 2. Registration and login

- [ ] 2.1 Add `workspace.CreateUserWithPassword(ctx, pool, email, passwordHash, name)` creating the user and personal Den in one transaction, and `workspace.FindUserByEmail(ctx, pool, email)`, with tests covering a fresh email succeeding, a duplicate email failing with a unique-violation, and a lookup miss
- [ ] 2.2 Add the `Register` handler: validate password length (≥8), hash with bcrypt, call `CreateUserWithPassword`, then `CreateSession` and set the session cookie exactly as `Callback` does, responding `204` on success; test covering success, duplicate email, and too-short password
- [ ] 2.3 Add the `Login` handler: look up by email, `bcrypt.CompareHashAndPassword` against the stored hash (or a fixed dummy hash when the email is unknown or the account has no `password_hash`, keeping the timing identical), then `CreateSession` and set the cookie; test covering correct credentials, wrong password, unknown email, and a Google-only account with no password set — confirming the last three return the same response
- [ ] 2.4 Wire `POST /auth/register` and `POST /auth/login` into `main.go` alongside the existing unauthenticated `/auth` routes, and verify both are reachable without a session and reachable requests to protected routes still 401 without one

## 3. Google becomes optional

- [ ] 3.1 Add `GET /auth/config` returning `{"googleEnabled": bool}`, true iff `GOOGLE_CLIENT_ID` is non-empty, and verify it flips with the env var
- [ ] 3.2 Verify the server boots and serves `/healthz` and password auth correctly with `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`GOOGLE_REDIRECT_URL` all unset (no crash, Google routes simply never succeed if hit)

## 4. Frontend

- [ ] 4.1 Add `register`, `login`, and `getAuthConfig` calls to the API client, and verify each against the running server
- [ ] 4.2 Build registration and login forms on the signed-out landing screen (toggle between them), using the existing design tokens and mascots per the visual system, and verify both submit and show field-level errors for a short password or a rejected login
- [ ] 4.3 Render the "Sign in with Google" control only when `getAuthConfig()` reports it enabled, and verify it's hidden with no Google env vars set and shown with them set
- [ ] 4.4 Route a successful register/login through the same "load current user, enter the Den" path the Google callback uses, and verify a fresh registration lands in an empty Den with no setup step, matching existing Google sign-up behavior

## 5. E2E coverage

- [ ] 5.1 Add a Playwright test that registers a new account through the real UI (not seeded — no external consent screen blocks this path) and lands in an empty Den
- [ ] 5.2 Add a Playwright test that registering with an already-used email shows an error and creates no account
- [ ] 5.3 Add a Playwright test that signing out and back in with the same email/password reaches the same Den with prior content intact

## 6. Documentation

- [ ] 6.1 Update `README.md`'s setup steps to note email/password sign-in needs no Google Cloud setup, and update `server/.env.example`'s comments to mark the `GOOGLE_*` variables optional
- [ ] 6.2 Re-verify the README's local-dev steps end to end using only password sign-in (no Google credentials configured), following it as someone unfamiliar with the project would
