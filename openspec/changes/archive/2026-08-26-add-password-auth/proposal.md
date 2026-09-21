## Why

`add-den-workspace` shipped with Google as the only sign-in method, which requires registering a domain and creating a Google Cloud OAuth client before the app is usable at all — that's blocking task group 1/13 (deploy) right now, and it's a real barrier to just running Hootden today. Email/password registration needs no external provider, no domain, and no OAuth console, so it unblocks using and hosting the app immediately while Google sign-in stays available for later.

## What Changes

- Add `POST /auth/register` (email + password + display name) and `POST /auth/login` (email + password), both issuing the same opaque session token and cookie the Google flow already issues — no new session mechanism, just a second way to obtain one.
- Add a `password_hash` column to `users`, hashed with bcrypt; `google_sub` becomes nullable so a password-only account has no provider identity, and vice versa.
- Enforce `email` uniqueness across all accounts regardless of how they signed up (currently unenforced, since Google's `google_sub` was the only real key). Registering with an email already in use — by either method — is rejected rather than merged; account linking is not in scope here.
- Frontend gains registration and login forms on the signed-out landing screen, alongside (not replacing) the existing "Sign in with Google" button, which only renders when Google OAuth is actually configured.
- **BREAKING**: `users.google_sub` drops its `NOT NULL` constraint and a new unique constraint is added on `users.email` — both are additive/loosening for existing rows, but any code assuming `google_sub` is always present needs updating (`workspace.UserResolver` and the auth middleware).

## Capabilities

### New Capabilities

(none — this extends the existing `accounts` capability rather than introducing a new one)

### Modified Capabilities

- `accounts`: adds a password-based sign-in/sign-up path as a second way to establish a session, alongside the existing Google OAuth requirement; adds an email-uniqueness rule that previously didn't apply. Note: `accounts` currently lives only in the not-yet-archived `add-den-workspace` change (`openspec/changes/add-den-workspace/specs/accounts/spec.md`) — this change's spec delta is written against that content and should land after `add-den-workspace` archives, or be reconciled at archive time if it hasn't yet.

## Impact

- **Server**: new `golang.org/x/crypto/bcrypt` dependency; new migration `0002_password_auth.sql`; new handlers in `internal/auth`; `workspace.UserResolver` and `auth.RequireAuth`'s account-lookup path need to tolerate a nullable `google_sub`.
- **Frontend**: signed-out landing screen (`web/app/page.tsx` or equivalent) gains register/login forms; API client gains `register`/`login` calls.
- **No change** to session issuance, the cookie shape, CORS, or CSRF posture — password auth is a second on-ramp to the same session mechanism, not a new one.
