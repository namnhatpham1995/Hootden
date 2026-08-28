## 1. Server-side registration input validation

Ordered first because it is independent of the form work and ships on its own — the frontend groups do not depend on it, and it closes the two gaps that make a bad registration permanent or unexplained.

- [x] 1.1 Reject a registration whose email is empty or not a well-formed address in `auth.PasswordHandlers.Register`, before hashing, and verify unit tests cover an empty string, an address with no `@`, an address with no domain, and a valid address still succeeding
- [x] 1.2 Reject a password over bcrypt's 72-byte limit with a 400 naming the limit, checked before `GenerateFromPassword` rather than by interpreting its error, and verify a 73-byte password returns 400 with a usable message while a 72-byte one still registers
- [x] 1.3 Verify no path in `Register` can still map a rejectable input to `500 registration failed`, by reviewing each error branch and confirming the 500 is reachable only for genuine infrastructure failures
- [x] 1.4 Verify `Login` is unchanged and still answers a malformed or over-long input identically to a wrong password, so the existing "unknown email is rejected the same way" test continues to pass

## 2. Labels and e2e locators

Ordered before the new field, and separately: it is a no-op refactor that can be reviewed and merged on its own, and it is what stops group 3 from breaking the suite. Doing it after would mean landing a red build and fixing it forward.

- [x] 2.1 Add a visible `<label>` bound to each existing input on the landing form, keeping the placeholders, and verify each field's accessible name survives typing (the name comes from the label, not the placeholder)
- [x] 2.2 Switch `web/e2e/password-auth.spec.ts` and any sibling spec that locates these fields from `getByPlaceholder` to `getByLabel`, and verify the full e2e suite passes unchanged against `docker compose --profile full`

## 3. Reveal control

- [x] 3.1 Add a single reveal/re-mask control governing every password field on the form, as a `<button type="button">` with an accessible name that states its action and a pressed state, and verify clicking it while the form is incomplete submits nothing
- [x] 3.2 Verify fields start masked on both sign-in and registration, that toggling changes only visibility and not the submitted value, and that reloading the form returns to masked

## 4. Confirmation field

Depends on group 2 for the locator change and group 3 for the shared reveal state.

- [ ] 4.1 Add a second password field rendered in registration mode only, labelled distinctly from the first, with `autoComplete="new-password"` on both, and verify sign-in mode still shows exactly one password field
- [ ] 4.2 Reject a mismatched pair before calling `register()`, showing the existing inline error and leaving the email and both entries as typed, and verify no request reaches the API on a mismatch
- [ ] 4.3 Clear the confirmation entry when switching between sign-in and registration, and verify switching modes never carries a stale confirmation into a fresh registration
- [ ] 4.4 Verify the API request body still carries exactly `{email, password}` with no third field

## 5. End-to-end verification

- [ ] 5.1 Add a Playwright test that a registration with mismatched entries shows the mismatch error, creates no account, and leaves the typed values in place, and verify it fails if the client-side check is removed
- [ ] 5.2 Add a Playwright test that toggling reveal exposes the typed password and submits the same value, covering both form modes
- [ ] 5.3 Verify registration, sign-out, and sign-in still work end to end with a password manager active in one browser, confirming the `type` toggle does not disrupt fill or save

## 6. Documentation

- [ ] 6.1 Check whether the README or `web/AGENTS.md` describe the sign-in form's fields or the e2e locator convention, and update whichever do — record a no-op check here if neither mentions them
