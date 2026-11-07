## Context

See proposal.md. Bun produces a stable `app.js` entry plus content-hashed split chunks. A stale entry can therefore reference a chunk removed by the latest image.

## Goals / Non-Goals

**Goals:** prevent stale entry/chunk mismatches and provide bounded recovery.

**Non-Goals:** retaining old release assets, changing API caching, or fixing unrelated browser transition errors.

## Decisions

Reference the entry module from the layout as `/assets/js/app.js?v=<digest>`, where the digest is derived from the served `js/app.js` when the server starts, and also serve it with `no-cache`. Staging verification showed that the Cloudflare zone's Browser Cache TTL rewrites the origin `no-cache` to `max-age=14400` for `.js` files, so revalidation headers alone cannot prevent a four-hour stale entry; a content-derived URL changes on every release that changes the entry, whatever an intermediary does to its cache headers. Serve hashed chunks as immutable only when the asset exists, so a missing chunk's 404 is never pinned by a browser or the edge. This preserves efficient cache use without retaining old assets. Add a single client-side recovery guard stored per page load; reload once on dynamic-import failure, then render a manual retry control. Retaining prior bundles was rejected because release artefacts are embedded in one image and would grow deployment storage indefinitely. Changing the Cloudflare Browser Cache TTL is documented as optional hardening in `docs/bot-protection-runbook.md`, not relied upon.

## Operational Decomposition

| Workstream    | Area                             | Coordination                                       |
| ------------- | -------------------------------- | -------------------------------------------------- |
| Asset headers | static asset handler             | Owns entry/chunk cache policy and tests.           |
| Recovery      | browser bootstrap                | Owns error detection and bounded reload behaviour. |
| Verification  | Go/JS tests and deployed headers | Runs after both workstreams.                       |

## Risks / Trade-offs

- [Entry revalidation increases a small request cost] → only the entry revalidates; chunks remain immutable.
- [Edge-cached catalogue HTML (up to five minutes) can still reference the previous entry URL] → the origin serves the current entry under any version query, and dynamic-module recovery bounds the remaining window.
- [Reload loop] → persist a one-time recovery guard and expose manual retry on recurrence.

## Migration Plan

Deploy normally. Pages rendered by the new release reference the new entry URL, so browsers fetch it on their next navigation; rollback restores the prior URL and cache policy.
