## Context

See `proposal.md` — Why. The existing `accounts` capability (defined in the not-yet-archived `add-den-workspace` change) establishes sessions as opaque server-side tokens delivered via an `HttpOnly` cookie, issued by `auth.CreateSession(ctx, pool, userID)` regardless of how `userID` was resolved. The Google flow resolves it via `workspace.EnsureUserAndDen`, which upserts on `google_sub` and treats sign-in/sign-up as one action. Password auth needs its own resolution path into that same `userID` → `CreateSession` → cookie pipeline; nothing about the session itself changes.

## Goals / Non-Goals

**Goals:**

- Register and sign in with email + password, with no external dependency beyond this repository.
- Reuse the existing session/cookie/CORS/CSRF machinery unchanged.
- Keep Google sign-in working, unmodified, for whoever configures it later.

**Non-Goals:**

- Password reset / forgot-password flow. Needs outbound email, which this repo has no infrastructure for yet. A password account with a lost password has no recovery path, same accepted risk the original design already takes for a lost Google account.
- Email verification at registration. Someone can register with an email they don't control; low stakes for a single-tenant personal app with no email-triggered action to abuse.
- Linking a Google account and a password account that share an email. Registering with an in-use email is simply rejected (see spec). Linking is real scope, deferred until it's actually needed.
- Rate limiting / brute-force protection on `/auth/login`. Worth having before this is exposed to strangers on the internet; not before. Called out below as an accepted risk, not silently skipped.
- A generic multi-provider auth abstraction (`AuthMethod` interface, provider registry, etc.). Two concrete flows funneling into one `CreateSession` call is simpler to read than a framework built for two implementations.

## Decisions

### bcrypt for password hashing

`golang.org/x/crypto/bcrypt`, cost factor 12. It's a new dependency, but the project's own stated exception is exactly this: migrations got `goose` because a hand-rolled runner risks corrupting state; password hashing gets `bcrypt` because a hand-rolled one risks the equivalent for credentials. `x/crypto` is the same origin as the `x/oauth2` dependency already in `go.mod`.

**Alternative rejected — argon2id.** OWASP's current first recommendation, and Go has `golang.org/x/crypto/argon2` too. It needs three tuned parameters (memory, iterations, parallelism) to actually be safer than bcrypt in practice; getting that tuning wrong is worse than bcrypt's one cost-factor knob. Bcrypt is OWASP's accepted second choice and is what the reference project (`personal-financial-management`, via Spring's default `PasswordEncoder`) uses too. Revisit if this is ever handling more than a personal workspace's worth of accounts.

### One nullable column each way, plus a check constraint

`users` gains `password_hash TEXT NULL`. `google_sub`'s existing `NOT NULL` is dropped. A `CHECK (google_sub IS NOT NULL OR password_hash IS NOT NULL)` constraint keeps every row authenticatable by at least one method — the database refuses an account nothing can sign into, rather than trusting every code path to maintain that invariant by hand.

`email` gets a `UNIQUE` constraint it didn't have before (it was display-only under Google-only auth, per the original design; password auth makes it a real lookup key). No existing rows can conflict — nothing is deployed yet, and the E2E seeding helper's emails already carry a random suffix.

### Two resolution functions, not one upsert

Google's `EnsureUserAndDen` upserts because sign-in and sign-up are the same action for it (there's no password to check, so "resolve or create" is safe and correct either way). Password auth can't collapse those the same way — registering must fail on a duplicate email, and signing in must verify a password that only exists on one of the two paths. So `workspace` gains two functions instead of overloading one:

- `CreateUserWithPassword(ctx, pool, email, passwordHash, name) (userID, denID, err)` — fails (unique violation) if the email is taken; creates the user + personal Den in one transaction, same shape as `EnsureUserAndDen`'s new-user branch.
- `FindUserByEmail(ctx, pool, email) (userID, passwordHash string, err)` — plain lookup for the login handler to verify against.

Both are called from new `Register`/`Login` handlers in `internal/auth`, which end the same way `Callback` does: `CreateSession` then `setSessionCookie`. Unlike `Callback` (a browser redirect target), these are `fetch`-called from the frontend forms, so they respond `204 No Content` on success (cookie set, same as `SignOut`'s response shape) and a JSON error body on failure, rather than redirecting.

### Login failure is one response, not two

Wrong password and unknown email return the identical `401` with the identical body. To keep that identical in *timing* as well as content — otherwise a fast "no such user" vs a slower "checked the hash and it didn't match" is itself a signal — the unknown-email path still runs `bcrypt.CompareHashAndPassword` against a fixed dummy hash before returning. One `if`, no measurable cost, closes a classic user-enumeration side channel for free.

### `GET /auth/config` tells the frontend whether Google is configured

The landing screen currently always renders a Google button. Once Google is optional, it needs to know whether to. A one-field, read-only, unauthenticated `GET /auth/config` → `{"googleEnabled": bool}` (true iff `GOOGLE_CLIENT_ID` is set) is simpler than threading that flag through the build, and it's a `GET` with no state to change, so it doesn't touch the CSRF posture (`design.md`'s "no state-changing `GET`" constraint applies to mutations, not to this).

## Risks / Trade-offs

**No rate limiting on `/auth/login`** → Accepted for now: this is aimed at self-hosting for one person or a small trusted group, not a public sign-up surface yet. Bcrypt's own cost factor is the only current friction against guessing. Add per-IP/per-account throttling before this is exposed publicly at any scale.

**No password reset** → Accepted, same shape as the existing "lost Google account = lost Hootden account" risk. Both get a real answer together once outbound email exists.

**A stray unverified email address just sits there** → Accepted; nothing in this change uses email for anything but display and login, so an unverified one causes no harm beyond looking odd to its owner.

## Migration Plan

Additive only, one goose migration (`0002_password_auth.sql`): add `password_hash`, drop `NOT NULL` on `google_sub`, add the check constraint, add the `email` unique index. No existing data to migrate — nothing is deployed yet, and every environment that does have rows (local dev, CI) already has unique, non-null emails.

`workspace.UserResolver.ResolveUser` (Google's path) and `auth.RequireAuth`'s session lookup are unaffected — both already key on `userID`/`google_sub`, never on the now-nullable column in a way that assumed it was always populated.
