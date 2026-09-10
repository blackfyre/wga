## 1. Shared cache concurrency

- [x] 1.1 Make the application cache loader coalesce concurrent misses per key while preserving generation-safe invalidation and uncached failures; verify with focused success, error, unrelated-key, and invalidation-race tests under `go test -race ./internal/utils`.

## 2. Results-only artwork search

- [x] 2.1 Extract one canonical artwork-results workflow and make the full search view compose it; verify focused workflow tests prove identical filters, result ordering, totals, pagination, canonical state, and cancellation checkpoints.
- [x] 2.2 Route `/artworks/results` through the results-only workflow without loading facet projections, while preserving full-page and `#artwork-search` responses; verify handler spies plus focused HTTP tests assert the required reads, fragment shape, `HX-Push-Url`, and direct non-HTMX fallback behaviour.
- [x] 2.3 Exercise the artwork-search HTMX contract in Chromium with `bunx playwright test playwright-tests/artwork-search.spec.ts playwright-tests/artwork-search-task71.spec.ts` against a running application and verify filtering, paging, URL updates, target swaps, and no-JavaScript submission remain functional.

## 3. Reusable artist availability

- [x] 3.1 Replace per-page artwork relation expansion with a compact application-scoped set of every artist ID referenced by published works, and verify repository tests cover empty data, unpublished works, duplicate relations, and artists appearing only as co-authors.
- [x] 3.2 Invalidate the artist-availability projection after relevant artwork saves and deletes using thin lifecycle hooks, and verify focused tests include mutation and invalidation-during-load cases without allowing an obsolete load to repopulate the cache.
- [x] 3.3 Use the reusable availability projection from the public artist index and both Dual Mode panes, and verify existing repository and Dual Mode tests plus a query-count assertion prove repeated page requests do not repeat the catalogue-wide relation query.

## 4. Reusable collection holdings

- [x] 4.1 Load collection holdings as one compact, complete counted projection and apply name filtering, deterministic ordering, the forty-option bound, selected-value retention, and omitted totals in memory; verify focused tests reproduce all existing venue-facet outputs for empty, searched, selected, truncated, and unknown selections.
- [x] 4.2 Invalidate collection holdings after relevant artwork, location, and artist lifecycle changes, and verify save/delete plus concurrent invalidation tests prove the next request observes current persisted labels, eligibility, and counts.
- [x] 4.3 Verify concurrent cold artwork searches share one collection projection load and subsequent venue queries reuse it without unbounded query-key growth by running focused handler and cache concurrency tests under the race detector.

## 5. Artist query indexes

- [x] 5.1 Capture `EXPLAIN QUERY PLAN` evidence for the supported published-artist name and birth ordering paths, select only indexes that remove demonstrated scans or temporary sorts, and record the exercised query shapes and database-size delta in the task completion evidence.
- [x] 5.2 Add the selected artist indexes through a timestamped reversible PocketBase migration and verify migration up/down tests prove exact index creation, rollback, existing-data preservation, and idempotent migration execution.
- [x] 5.3 Run focused artist repository tests for ascending name, descending name, birth order, paging, letter, school, period, and born-range filters, and confirm post-migration query plans use the intended indexes without changing returned IDs or order.

## 6. Artwork list/count experiment

- [x] 6.1 Compare the existing separate artwork list/count queries with a combined window-count candidate for unfiltered, text, exact-artist including co-authors, venue, and date-range cases; retain the combined production query only if outputs are identical and representative CPU/allocation benchmarks improve without a material supported-case regression, otherwise leave production behaviour unchanged and report the rejection evidence.

## 7. Final performance and regression verification

- [x] 7.1 Repeat the production-shaped concurrent profile matrix for `/artworks`, `/artworks/results`, and `/dual-mode`, recording dataset cardinality, request concurrency, throughput, latency, CPU samples, allocations per request, forced-GC retained heap, and dominant call paths; verify removed catalogue-wide work no longer appears per request and investigate any material regression before completion.
- [x] 7.2 Run formatting, `go vet ./...`, `go test ./... -cover`, and the focused Playwright artwork-search checks, then verify the final diff contains no startup-generation, Cloudflare, external-cache, admission-threshold, public-route, or unrelated runtime-tuning changes.
