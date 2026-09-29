## Context

Production traces now show `/dual-mode` spending about half its average duration in a reference stage that sequentially reloads schools, periods, and artist birth-year bounds for every request. The same traces show artwork-search facet construction dominating `/artworks`, but the current facet span combines cached taxonomy options, venue projection work, and two grouped count queries.

WGA already has application-scoped, coalesced cache helpers and mutation hooks for catalogue projections. Repository packages own reusable persisted-data projections; handlers remain adapters and view assemblers. Existing trace helpers enforce typed names and bounded attributes.

## Goals / Non-Goals

**Goals:**

- Remove repeated Dual Mode reference queries after one successful cold load.
- Invalidate the reference projection immediately after relevant successful mutations.
- Preserve request cancellation and share concurrent cold loads.
- Make the artwork facet bottleneck attributable to one bounded substage in production traces.

**Non-Goals:**

- Rewriting grouped facet SQL without query-plan and substage evidence.
- Caching arbitrary visitor filter combinations.
- Loading facets through a separate HTMX request or changing their presentation.
- Changing catalogue URLs, results, ordering, pagination, or Dual Mode behaviour.

## Decisions

### Own the Dual Mode reference projection in the repository layer

Add a repository-owned projection containing the stable school, period, and birth-bound source data required by Dual Mode. Its context-aware loader will use the existing instrumented cache primitive with a new allow-listed cache identity. Dual Mode will transform the returned immutable data into its local lookup maps.

This keeps mutation hooks from importing an HTTP handler package and moves non-trivial persisted-data composition out of the handler. Keeping the cache inside the handler was rejected because it would invert the hook-to-feature dependency and preserve business workflow in a framework adapter.

### Use mutation invalidation rather than a time-only expiry

Successful artist, school, and art-period creates, updates, and deletes will invalidate the projection generation. Concurrent requests that overlap invalidation retain their completed generation privately, while the next request performs one shared fresh load. Returned slices and maps will be cloned or reconstructed so callers cannot mutate cached state.

A short TTL alone was rejected because it knowingly serves stale administrative edits. A bounded TTL may remain as defence in depth only if it does not replace mutation invalidation.

### Add typed child stages beneath the existing facet workflow span

Retain `wga.workflow.artwork_search.facets` as the coarse parent and add allow-listed child stages for option loading, venue loading, school counts, and form counts. Each child records only its stable name and bounded outcome. Context-free compatibility paths remain telemetry no-ops, and cancellation flows through the child context.

Adding SQL text, filter state, option names, or record counts as span data was rejected because it conflicts with the existing privacy and cardinality contract.

### Defer facet execution changes until the new evidence is deployed

The school and form counts already use one grouped `COUNT(DISTINCT ...)` query per facet, an earlier optimisation that materially reduced latency. This change will not combine, parallelise, cache, or defer those queries without knowing which substage dominates current production work.

The next optimisation decision will use the refined traces and `EXPLAIN QUERY PLAN` evidence. This avoids increasing SQLite contention through speculative concurrency or creating an unbounded filter-result cache.

## Risks / Trade-offs

- [Risk] Missing a mutation source could serve stale Dual Mode reference data. → Bind focused hook tests for all three source collections and retain cold-load integration coverage.
- [Risk] Cached mutable structures could leak mutation between requests. → Store value projections and return defensive copies or reconstruct handler lookup maps.
- [Risk] New spans increase trace volume. → Add only four bounded stages beneath full artwork-search requests; results-only fragments continue to skip facet work.
- [Risk] The refined traces may show several similarly expensive stages. → Treat that as evidence for a later query-plan change rather than broadening this implementation.

## Migration Plan

Deploy as an application-only change with no persistence or configuration migration. Observe cache events and refined facet traces in UAT before production promotion. Rollback restores per-request reference loading and the previous coarser trace shape without affecting stored data or public behaviour.
