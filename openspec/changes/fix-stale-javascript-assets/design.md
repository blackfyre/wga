## Context

See proposal.md. Bun produces a stable `app.js` entry plus content-hashed split chunks. A stale entry can therefore reference a chunk removed by the latest image.

## Goals / Non-Goals

**Goals:** prevent stale entry/chunk mismatches and provide bounded recovery.

**Non-Goals:** retaining old release assets, changing API caching, or fixing unrelated browser transition errors.

## Decisions

Serve `app.js` with `no-cache` so browsers revalidate it on each navigation, while serving hashed chunks as immutable. This preserves efficient cache use without retaining old assets. Add a single client-side recovery guard stored per page load; reload once on dynamic-import failure, then render a manual retry control. Retaining prior bundles was rejected because release artefacts are embedded in one image and would grow deployment storage indefinitely.

## Operational Decomposition

| Workstream | Area | Coordination |
|---|---|---|
| Asset headers | static asset handler | Owns entry/chunk cache policy and tests. |
| Recovery | browser bootstrap | Owns error detection and bounded reload behaviour. |
| Verification | Go/JS tests and deployed headers | Runs after both workstreams. |

## Risks / Trade-offs

- [Entry revalidation increases a small request cost] → only the stable entry revalidates; chunks remain immutable.
- [Reload loop] → persist a one-time recovery guard and expose manual retry on recurrence.

## Migration Plan

Deploy normally. Browsers revalidate `app.js` on their next navigation; rollback restores the prior cache policy.
