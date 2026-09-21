## Why

The signed-out landing screen asks for a password you cannot see and, when registering, cannot check. A typo in a masked field is invisible at the moment it is made and only surfaces later as a sign-in that will never succeed — and because the account has no password reset and no second identity method configured (`/auth/config` currently reports `googleEnabled: false`), a mistyped password at registration means the account is unrecoverable. The person has no way to discover this: registration succeeds, and the wrong password is the one that was stored.

Revealing the password on demand and confirming it once at registration are the two standard mitigations, and neither exists today.

## What Changes

- Add a control that reveals and re-masks the password as it is typed, on both the sign-in and registration forms.
- Add a second password field to the registration form only, and reject a registration whose two entries differ before anything is submitted.
- Require the submitted email address to be a well-formed address at the API, not only in the browser. `Register` currently validates password length but passes the email straight to the insert, so the browser's `type="email"` attribute is the only thing standing between a malformed address and a permanently unreachable account.
- Reject a password longer than bcrypt accepts, as a stated limit rather than an internal error. `golang.org/x/crypto@v0.55.0/bcrypt` returns `ErrPasswordTooLong` above 72 bytes, and `Register` maps every error from hashing to `500 registration failed` — so a passphrase from a password manager can be refused with nothing the person can act on. This lands here rather than in its own change because it is the same handler, the same validation pass, and the same spec requirement as the email check.

Not changing: password strength rules, the 8-character minimum, password reset (there is none, and adding one is a larger change), or anything about session issuance.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accounts`: adds a requirement that a password being entered can be revealed at the person's request; and modifies "Register with email and password" so registration requires the password twice and rejects a malformed email address.

## Impact

- **`web/components/SignedOutLanding.tsx`.** The form gains a reveal control, a conditional second field, and per-field labels. Labels are not cosmetic here — the form currently identifies its inputs by placeholder alone, so the accessible name disappears the moment someone starts typing, and a second field placeholdered "Confirm password" makes the ambiguity worse rather than better.
- **`web/e2e/password-auth.spec.ts` (and any sibling spec locating these fields).** `getByPlaceholder("Password")` matches case-insensitive substrings, so "Confirm password" also matches it and the locator resolves two elements — a strict-mode violation failing all three password-auth tests. This breaks the moment the field is added, so the locator change ships in the same PR as the field.
- **`server/internal/auth/password.go`.** `Register` gains email validation and a maximum password length, both as explicit rejections before hashing. `Login` is untouched: it must keep treating every failure identically, so a malformed or over-long input there is still an ordinary "invalid email or password".
- **No impact** on the database schema, session handling, the API's request or response shapes, or `Login`'s behaviour. The confirmation field is never transmitted.
