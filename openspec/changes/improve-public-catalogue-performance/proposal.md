## Why

Admitted catalogue and Dual Mode requests still perform avoidable full-dataset SQLite work, duplicate cold-cache loads, and substantial allocation churn. Production-sized profiling now identifies specific request paths where reducing cost per request can improve visitor latency and Railway headroom without broad response caching or additional infrastructure.

## What Changes

- Make artwork results-only requests load and project only the result data required by their returned fragment.
- Replace repeated full-artwork author-availability scans with a bounded, co-author-correct reusable projection.
- Make collection-holding options a bounded reusable projection and suppress duplicate work during concurrent cold loads.
- Add database indexes that match supported public artist ordering and filtering paths where query-plan evidence demonstrates a benefit.
- Evaluate consolidation of duplicate artwork list/count work and retain it only when representative profiling demonstrates an improvement.
- Establish before-and-after query-plan, allocation, and concurrency evidence for each optimisation while preserving existing catalogue, paging, and Dual Mode behaviour.
- Keep startup/background publication generation, Cloudflare configuration, external caches, and unrelated runtime tuning outside this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: Require results-only interactions and public catalogue reads to avoid unnecessary catalogue-wide work while preserving existing filtering, co-author, paging, and Dual Mode semantics.

## Impact

- Affects artwork search workflows and fragments, artist-index repositories, Dual Mode artist availability, application-scoped reference caching, cache invalidation hooks, and PocketBase schema indexes.
- Adds focused query-plan, concurrency, allocation, repository, handler, and HTMX-contract verification.
- Does not change public routes, URL parameters, rendered result semantics, deployment topology, or external dependencies.
