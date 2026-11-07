## Evidence and attribution

Sentry WGA-BROWSER-5 (7698114864), staging event
`220f5b8a64284adcaceb3c36e8225b55`, occurred on 28 August 2026 at
16:58:57.584 UTC in Opera 134 on Windows. The issue is already resolved;
this investigation does not change Sentry settings or status.

The event has no stack, raw stack, or release, so source-mapped attribution
is unavailable. Breadcrumbs show typing in `#artwork-query`, repeated aborted
`GET /artworks` requests, then a successful response at 16:58:57.250 and an
`/artworks` history update at 16:58:57.539. There is no evidence of a dialog
operation immediately before the error.

Attribution to the artwork-search HTMX View Transition path is an inference
from those breadcrumbs and source inspection, not a recovered historical stack.
The coordinator explicitly authorised deterministic reproduction with this limit.
The exact historical invalid-state trigger remains unknown.

The current path is `#artwork-filters` -> `GET /artworks` ->
`searchWithCheckpoint` -> `ArtworkSearchBlockResponse` -> outer swap of
`#artwork-search`. The layout enables global View Transitions. HTMX 2.0.10
calls `document.startViewTransition`, passing its swap and settle promise,
but discards the returned object, leaving `ViewTransition.ready` unobserved.

## Decision

Wrap the native entry point once in the browser entry module, before importing
HTMX bootstrap. Observe only `ready`; swallow only a native DOMException named
InvalidStateError after `updateCallbackDone` succeeds. Return the original
transition unchanged. Let native skipped-animation semantics execute the swap
once; never retry the DOM callback or network request. Callback failures and
other rejection types stay observable. Browsers without the API retain the
existing HTMX swap path. Do not filter global errors or change dialog behaviour.

## Reproduction and verification

A Chromium Playwright test inserts two rendered elements with the same
`view-transition-name`, then performs two real artwork searches. Native snapshot
capture rejects `ready` with InvalidStateError while both result and history
updates complete. Without the guard this fails with two unhandled errors:
“Transition was aborted because of invalid state. Snapshot capture failed”.
This is a deterministic reproduction of the API/path failure, not proof that
historical duplicate names caused WGA-BROWSER-5.

Browser coverage also exercises successful transitions, missing API, unrelated
rejections, callback failures, and unrelated global InvalidStateError. Required
Go, asset, lint and browser checks are tracked in tasks.md.

## Alternatives

Disabling transitions globally would remove working navigation animation.
A global unhandled-rejection filter would conceal unrelated defects. Neither is
necessary for this promise-specific failure.
