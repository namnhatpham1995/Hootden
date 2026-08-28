## 1. Failure-state test harness

Ordered first because every group below needs it, and because three of these bugs exist precisely for want of a test that can fail. Landing it first means each subsequent group ships with proof rather than an assertion that it works.

- [x] 1.1 Add an e2e helper that forces a chosen status on a matching API route via Playwright request interception, and verify it by making an existing save fail and observing the editor's "Failed to save" state
- [x] 1.2 Add an e2e helper that revokes the signed-in session server-side mid-test, reusing the direct-Postgres approach in `web/e2e/db.ts`, and verify a following API call receives a genuine 401 rather than a simulated one

## 2. Save failures are classified

Independent of the navigation work and shippable alone; the retry loop stops lying about unsaveable documents even before the flush exists.

- [x] 2.1 Branch `runSave`'s failure handling on `ApiError.status` per design.md — 5xx and network failures retry, other 4xx stop — and verify unit-level coverage that a 400 schedules no retry and a 503 does
- [x] 2.2 Add a terminal "could not be saved" status distinct from "retrying", showing why, and verify the editor's content is still present and editable in that state
- [x] 2.3 Add an e2e test using the 1.1 helper that a persistently-400 save stops retrying, reports itself, and leaves the typed text in the editor, and verify it fails when the classification is reverted

## 3. Session loss during an edit

- [x] 3.1 Invert `apiFetch`'s 401 handling so the redirect is requested by callers that want it rather than escaped by callers that do not, fold `getCurrentUser`/`postAuth`'s bare `fetch` calls back onto the shared path, and verify every existing e2e test still passes
- [x] 3.2 Handle a 401 from `saveDoc` by stopping the retry loop and showing that the session ended, without navigating, and verify with the 1.2 helper that the typed text is still on screen after a mid-edit revocation
- [x] 3.3 Verify the revoked-session case no longer produces a repeating leave-confirmation prompt, by revoking mid-edit and confirming no dialog recurs over at least two backoff intervals

## 4. Pending saves survive leaving the editor

Depends on group 2 for the failure classification the flush reports through.

- [x] 4.1 Flush a pending or retrying save in `Editor`'s unmount cleanup instead of clearing the timer, and verify the request is issued and completes after the component has unmounted
- [x] 4.2 Warn before discarding changes that could not be flushed, and verify the warning appears for a flush that fails and not for one that succeeds
- [x] 4.3 Add an e2e test that typing and immediately selecting another page persists the edit — no waiting for "Saved" — and verify it fails against the current `clearTimeout` behaviour
- [x] 4.4 Add an e2e test that leaving during a retry backoff either saves or warns, using the 1.1 helper to fail the first save, and verify the work is never dropped silently

## 5. Sign-in check has a failure state

- [x] 5.1 Give `useDen`'s `me` an explicit error state distinct from loading and signed-out, and verify `Den` no longer renders `null` when `getCurrentUser` rejects
- [x] 5.2 Render an "account could not be loaded" state with a retry action, and verify it is shown for a 500 from `/me` and never for a 401
- [x] 5.3 Add an e2e test using the 1.1 helper that a 500 from `/me` shows that state rather than the landing screen or a blank page, and verify a genuine signed-out load still reaches the landing screen

## 6. Documentation

- [ ] 6.1 Update `web/AGENTS.md` (or `web/CLAUDE.md`, whichever documents the API client) to record that the 401 redirect is opt-in and why, so the next write endpoint does not rediscover this — and verify the note names `saveDoc` as the case that motivated it
- [ ] 6.2 Check whether the README or e2e documentation describes the test suite's capabilities, and add the failure-injection helpers if so, recording a no-op check here if not
