## Context

See `proposal.md — Why`. This is a small, contained change; the design notes exist for three decisions that are easy to get subtly wrong, not because the shape is in doubt.

Current state: `web/components/SignedOutLanding.tsx` is one client component holding both forms, toggled by a `mode` state, styled with inline objects against the project's design tokens. It has no `<label>` elements — inputs are identified by placeholder alone. `server/internal/auth/password.go`'s `Register` validates password length and nothing else.

## Goals / Non-Goals

**Goals:**

- Reveal and confirmation behave the way people already expect from other sign-up forms, with no novelty to learn.
- The e2e suite still passes, and keeps testing the same things.

**Non-Goals:**

- No password strength meter, no rules beyond the existing 8-character minimum.
- No password reset. Its absence is what makes a registration typo unrecoverable and therefore motivates this change, but building it is a separate, larger change.
- No redesign of the landing screen.

## Decisions

### One reveal toggle, not one per field

The registration form has two password fields whose entire purpose is to hold the same value. Revealing one and not the other lets a person compare a visible string against a masked one, which is exactly the comparison they were trying to avoid making by eye. A single control governing both is less code and the more useful behaviour.

The control must be `<button type="button">`. A bare `<button>` inside a `<form>` defaults to `type="submit"`, so peeking at the password would fire a sign-in attempt. The existing mode-toggle button already does this correctly and is the pattern to copy.

**Alternative rejected — a checkbox labelled "Show password".** Functionally fine and slightly simpler, but it reads as a form field with a value to submit rather than an action, and it takes a full row in a narrow column that currently has none to spare.

### Confirmation is checked in the browser and never transmitted

The API keeps receiving `{email, password}`. A second field in the request body would be a value the server must compare and could disagree about, for no gain: the server cannot detect a typo, because both entries would be the typo.

This does mean the API is not the enforcement point for this rule, which is a deliberate exception to validating at the trust boundary. It is safe precisely because there is nothing to enforce — a caller bypassing the browser submits one password, and one password is what registration has always accepted. Nothing is weakened by its absence. Email well-formedness is the opposite case and does move to the server, because a malformed address there produces a real, permanent artefact: an account nobody can reach.

**Alternative rejected — send both and compare server-side.** Adds a field, a rejection path, and a test for a mismatch that a correct client can never produce.

### Check the password length before hashing, not by interpreting the hasher's error

`bcrypt.GenerateFromPassword` returns `ErrPasswordTooLong` above 72 bytes, so the fix could be a branch on that error. Checking the length first is better for two reasons: it puts the limit next to the existing minimum-length check where anyone reading the validation sees both, and it does not make the API's contract depend on which error value a dependency happens to return — older `x/crypto` truncated silently instead of erroring, and a future version could change again.

The limit is bcrypt's, so it belongs in a named constant beside `minPasswordLength` rather than inline, and the rejection message should state it. 72 *bytes*, not characters: a passphrase with accented or non-Latin characters reaches the limit sooner than its length suggests, which is worth saying in the message rather than leaving someone to guess.

**Alternative rejected — raise the ceiling by pre-hashing with SHA-256 before bcrypt.** The standard way to remove the limit entirely, and wrong here: it changes the stored hash format, so it needs a migration path for existing accounts, to solve a problem nobody has reported.

### Locate e2e fields by label, not placeholder

Adding a second field breaks the existing tests before it breaks anything else. `page.getByPlaceholder("Password")` matches case-insensitive substrings, so "Confirm password" also matches and the locator resolves two elements — a Playwright strict-mode violation that fails all three password-auth specs.

Two ways to fix it. `{ exact: true }` is the one-word change, but it leaves the tests coupled to display copy and will break again the next time the wording moves. Adding real `<label>` elements and switching to `getByLabel` fixes the tests and the accessibility gap in one move: a placeholder is not an accessible name, and it disappears the moment someone types, so a screen-reader user reviewing a filled form has no way to tell which field is which. With two nearly identical password fields, that is not a hypothetical.

Labels are therefore load-bearing here, not polish, and they ship with the field rather than after it.

## Risks / Trade-offs

**Revealed passwords are readable by anyone near the screen** → Inherent to the feature and the reason it is opt-in per entry rather than remembered. The spec requires the revealed state not to outlive the entry, so a reload or a return visit starts masked.

**Password managers can behave oddly when a field's `type` changes** → Some fill heuristics key off `type="password"`. Mitigated by keeping `autoComplete` correct and unchanged on every field (`new-password` on both registration entries, `current-password` on sign-in), which is what managers actually key off, and by verifying fill and save still work in one browser with a manager active.

**Two more inputs on a 320px-wide column** → The form is already vertical and narrow. The reveal control sits inside the field rather than beside it, so it costs no row; the confirmation field costs one, and only in registration mode.
