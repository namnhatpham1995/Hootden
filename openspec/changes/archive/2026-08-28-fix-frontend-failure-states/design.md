## Context

See `proposal.md — Why` for the four failure states and how each behaves today.

What shapes the approach:

- Three of the four are violations of requirements the `pages` spec already carries. The specification was right; nothing executable could observe it being broken. Fixes therefore need tests that can fail, not just code.
- `apiFetch`'s 401 redirect is correct for reads and wrong for writes, and its own comment explains why it was written that way. Four of its call sites now want something else.
- `Editor` is deliberately remounted per page (`key={id}`), with a comment calling that "simpler and safer than resetting the save/retry refs by hand". That decision is right and stays; the unmount path is what needs work.
- The e2e suite has never made the API fail. Every existing test runs against a healthy stack.

## Goals / Non-Goals

**Goals:**

- No path where typed text disappears without the person being told.
- Each failure state is reachable in a test, so a regression fails the build rather than reaching a user.

**Non-Goals:**

- No change to the 800 ms debounce, the backoff curve, or the 1 MB body limit.
- No cross-tab coordination. The `pages` spec accepts last-write-wins for one account in two tabs, and this change does not reopen it.
- No offline queue or local draft persistence. Keeping text on screen and telling the truth about its state is the requirement; surviving a browser crash is a larger change.

## Decisions

### Flush on unmount rather than block the navigation

Two ways to stop a page switch dropping the buffer: prevent the navigation until the save finishes, or let it proceed and fire the save on the way out.

Blocking means intercepting `router.push` from inside `Editor`, which the component does not own — navigation is triggered by `PageTree` through `Den`'s `onSelect`. Threading a "can I leave?" negotiation up through both is a large change to two components for one caller's benefit, and it makes every page switch feel slower to prevent a loss that only matters when a save is genuinely pending.

Flushing on the way out is local to `Editor`: the unmount cleanup already runs, and it can fire the save instead of cancelling it. The request outlives the component — an in-flight `fetch` is not cancelled by unmounting, which is why in-flight saves already survive today while scheduled ones do not.

The residual case is a flush that itself fails, with the component already gone and nothing left to retry with. That is the case the spec requires a warning for, and it is the reason the warning cannot simply be deleted once flushing exists.

**Alternative rejected — persist the buffer to `localStorage` on unmount and restore it.** It survives more (a crash, a closed tab) but introduces a second source of truth for document content, with its own staleness and conflict questions against a server copy that may have moved on. Too large for the problem, and it would need its own decision about what wins on restore.

### `saveDoc` stops inheriting the 401 redirect; the redirect becomes opt-in

`apiFetch`'s auto-redirect now has four call sites that opt out — `getCurrentUser`, `login`, `register`, and `saveDoc` — against the reads that want it. A default that most deliberate callers override is the wrong default.

Inverting it means the redirect is requested by the calls that want it rather than escaped by the calls that do not, and the escape hatches (`postAuth`, the bare `fetch` in `getCurrentUser`) collapse back into the shared path. That removes the duplicated fetch plumbing those helpers exist to provide.

**Alternative rejected — leave the default and add a fourth exception.** Smallest diff today. But each exception has been a bug discovered after the fact rather than a decision made up front, and the next write endpoint gets the wrong behaviour by default too.

### Failures are classified by status, not by message

The retry loop needs to know whether another attempt could succeed. HTTP status is already the answer, and `ApiError` already carries it:

```
  401                    → session ended. Stop. Keep the text, say they were signed out.
  4xx (400, 404, 413…)   → this request will never succeed. Stop. Report it.
  5xx, network failure   → could succeed later. Retry with the existing backoff.
```

`ApiError` exists and carries `status`; a network failure surfaces as a `TypeError` from `fetch` and is not an `ApiError` at all, which distinguishes it without any extra plumbing.

**Alternative rejected — retry everything, with a cap on attempts.** Simpler branch, but it makes an unsaveable document look like a network problem for however many attempts the cap allows, and the person is told "retrying…" the whole time. The distinction is the useful part.

### `me` gains a third state rather than a second

`useDen` returns `me` as `Me | null | undefined`, where `undefined` means "still loading". `Den` renders `null` for it. An error currently collapses into that same `undefined`, which is why the page stays blank forever.

Adding an explicit error value keeps the existing two branches meaning exactly what they mean today and gives the third case somewhere to go. The alternative — a separate `error` field alongside `me` — permits states that cannot occur (loaded *and* errored) and pushes the ambiguity into every consumer.

## Risks / Trade-offs

**A flush on unmount can still fail, and there is no component left to retry it** → Which is precisely when the spec requires a warning, so the warning stays. Practically this means the person may be told after the fact that a page switch lost work — worse than not losing it, better than today's silence.

**`beforeunload` cannot be made reliable** → It is best-effort by design: browsers suppress the prompt without prior interaction, and it cannot await an async save. It stays as the last line for full-page unloads, and the unmount flush is what actually carries the guarantee for in-app navigation.

**Inverting the 401 default touches every API call site** → Mechanical but broad, and a missed read call site fails quietly by not redirecting rather than loudly. Mitigated by it being a compile-time signature change rather than an optional argument, so every call site must be visited.

**Testing failure states needs the suite to break the API on purpose** → New capability for this suite: intercepting requests to force a status, and revoking a session mid-test. Route interception covers the response cases; the 401 case is more honest driven through a real revocation, since that is what actually happens in production.
