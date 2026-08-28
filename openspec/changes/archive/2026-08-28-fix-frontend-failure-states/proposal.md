## Why

The frontend handles four failure states by silently doing nothing, and three of them lose the person's work. All four are invisible on localhost and structurally invisible to the e2e suite, which runs against a healthy API inside a five-second session.

**Navigating between pages discards a pending save.** `Editor` is mounted with `key={id}` and the sidebar navigates with `router.push`, so selecting another page fully remounts it. The unmount cleanup runs `clearTimeout(saveTimer.current)`, cancelling whatever save was scheduled. A client-side navigation fires no `beforeunload`, so the existing warning never appears either. The window is the 800 ms debounce at best — and up to the 16-second retry backoff when a save has already failed, which is exactly when the screen is displaying *"Failed to save — retrying…"* over work that is about to be dropped.

**A 401 during a save discards the buffer and then loops.** `apiFetch` reacts to any 401 by setting `window.location.href = "/"`. From a read that is correct and the code says so. From `saveDoc` it navigates away from unsaved content; and because `runSave`'s `catch` has already scheduled a retry, cancelling the browser's leave-confirmation prompt just means the next retry triggers it again, every 16 seconds, with no way to save and no way to stop.

**A non-retryable save failure retries forever.** `runSave` treats every rejection identically. A document exceeding the 1 MB body limit is rejected as `400 invalid request body` on every attempt, so the editor retries a request that cannot ever succeed while showing "retrying…" indefinitely.

**A 5xx from `/me` renders a blank page.** `getCurrentUser` throws on any non-401 error, `useDen` has no `.catch`, so `me` stays `undefined` and `Den` returns `null`. A database blip or a container restart during load gives a white screen with an unhandled rejection in the console — no error, no retry, no prompt to reload.

The first two are not new requirements. The `pages` spec already states that a failed save "MUST keep the unsaved changes in the editor and retry rather than discarding them silently", and already requires a warning when someone navigates away with unsaved changes. The implementation reads "navigate away" as leaving the site, which is where the gap opened.

## What Changes

- Flush a pending save when the editor unmounts, so switching pages persists what is in the buffer instead of cancelling it, and warn before discarding anything that cannot be flushed.
- Stop `saveDoc` inheriting the shared 401-redirect. A write that fails on an expired session must surface as a recoverable state that keeps the text on screen, not a navigation away from it.
- Distinguish retryable from non-retryable save failures. Retry connectivity and server errors; stop on a rejection the same request will always receive, and say so.
- Give the sign-in check a failure state distinct from "signed out", so a `/me` error renders something a person can act on rather than an empty document.

Not changing: the save cadence, the retry backoff curve, the 1 MB body limit, or last-write-wins across tabs — the `pages` spec already accepts that resolution and this change does not revisit it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pages`: modifies "Edits save automatically" so that moving between pages inside the app counts as navigating away, and adds the requirement that a save which cannot succeed on retry stops and reports itself instead of retrying forever.
- `accounts`: adds a requirement that a failure to determine whether someone is signed in is reported to them, and is not presented as either a signed-out state or an empty page.

## Impact

- **`web/components/Editor.tsx`.** Unmount flush, and a retry loop that branches on the failure. The `beforeunload` guard stays for full-page unloads.
- **`web/lib/api.ts`.** `saveDoc` stops routing through the shared 401 handler, joining `getCurrentUser`, `login`, and `register`, which already opt out for their own reasons. With four of its call sites now opting out, whether the redirect should stay the default is worth settling here rather than adding a fourth exception.
- **`web/lib/useDen.ts` and `web/components/Den.tsx`.** `me` gains a distinguishable error state; `Den`'s `me === undefined` branch stops meaning both "still loading" and "the request failed".
- **`web/e2e/`.** New coverage for each failure state. These need the API to fail on demand — an unreachable origin, a revoked session, an oversized document — which is a shape the current suite has no example of.
- **No server-side impact.** Every one of these is a client reaction to a response the API already returns correctly.
