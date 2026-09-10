## Context

See `proposal.md` for motivation. WGA is a single PocketBase process backed by SQLite. Profiling against 5,829 artists and 52,866 artworks showed SQLite consuming 95–97% of CPU on concurrent `/artworks`, `/artworks/results`, and `/dual-mode` requests, while forced-GC profiles retained almost no route-owned heap.

Three avoidable paths dominate. The results-only artwork endpoint builds the complete filter/facet view before rendering only its results. Artist availability expands every published artwork's JSON author relation for each displayed artist page. Collection options aggregate the full published catalogue and concurrent cold cache misses can execute the same loader repeatedly because the current cache helper does not coalesce in-flight loads. The default artist order also lacks an index over `filing_name`.

The authoritative catalogue specification requires exact artist matching to include co-authored works and requires unchanged filtering, URL, paging, no-JavaScript, and Dual Mode behaviour. An optimisation that treats only the first author as authoritative is therefore invalid even if it produces a faster query.

## Goals / Non-Goals

**Goals:**

- Remove data work that is not represented in the requested artwork-search fragment.
- Make repeated catalogue-wide reference derivations bounded, shared, and safely invalidated.
- Preserve co-author-correct artist availability and current collection-holding semantics.
- Add only indexes whose supported query plans and representative measurements demonstrate value.
- Compare before-and-after CPU, allocations, query plans, and concurrent cold-load behaviour.

**Non-Goals:**

- Caching personalised full-page or HTMX HTML.
- Adding Valkey, Redis, another service, or another deployment replica.
- Optimising startup sitemap or agent-publication generation.
- Changing public filters, routes, URL state, page size, result order, or admission thresholds.
- Tuning Go or SQLite memory settings before request work is reduced and re-profiled.

## Decisions

### Split results projection from full search-view assembly

The artwork search capability will expose a results workflow that parses the same canonical filter state and loads only the result count, requested records, result projection, and pagination data. Full-page and complete search-block responses will compose that results workflow with filter-option and collection projections. The handler will select the response contract from the route and validated HTMX target, invoke the matching workflow, and render its fragment.

This preserves one source of truth for result semantics while ensuring `/artworks/results` cannot accidentally pay for facets it does not return. Keeping one monolithic builder with skip flags was rejected because it makes invalid field combinations representable and obscures which reads each fragment requires.

### Cache complete bounded projections, not arbitrary request responses

Artist availability will be represented by the complete set of artist IDs occurring anywhere in published artwork author relations. It will be loaded once per cache generation, reused by artist and Dual Mode page projections, and intersected with each bounded candidate page in memory. This preserves co-authored works while changing a catalogue-wide expansion from per request to per generation.

Collection holdings will likewise be cached as the complete counted location projection. Query-specific name filtering, deterministic ordering, the forty-option limit, selected-value retention, and omitted-holdings accounting will run against that bounded projection in memory. Arbitrary query strings and rendered responses will not become cache keys.

An indexed first-author expression was rejected for artist availability because it violates co-author semantics. A normalised artwork-author table was deferred because the current mostly read-only catalogue does not yet justify a new persisted relation, import contract, and reconciliation workflow; it remains an alternative if bounded projection loading later proves too expensive.

### Coalesce cold loads in the shared cache primitive

The application cache helper will guarantee one in-flight loader per application and cache key. Concurrent callers will wait for that load and receive the same value or error. Invalidation will advance the existing generation before removing a value; a load begun under an obsolete generation must not republish stale data. The helper will not retain failed values, and unrelated keys will remain independent.

Changing only the venue caller was rejected because artist availability needs the same guarantee and the existing generic helper already owns cache-generation concurrency. The change must include deterministic concurrency and invalidation-race tests before callers adopt it.

### Invalidate projections from owning record lifecycle hooks

Artwork changes affecting publication, authors, or current location invalidate both reusable projections. Location changes affecting identity or display name invalidate collection holdings. Artist deletion or restoration invalidates collection holdings because the existing collection aggregate counts works only when its first author exists. Invalidation may conservatively occur for all saves in these owning collections when field-level change detection would be less reliable; correctness takes priority over avoiding a cheap cache eviction.

Hooks will only invalidate application cache state. Projection queries and transformation rules remain in the owning repository/workflow packages rather than lifecycle adapters.

### Add query indexes as reversible PocketBase migrations

A timestamped migration will add indexes matching supported published-artist orders, beginning with `(published, filing_name, id)` and adding the birth-year order only if `EXPLAIN QUERY PLAN` and representative benchmarks show it removes a scan or temporary sort. The migration will use the repository's PocketBase collection-index conventions and provide a down path that removes only its own indexes.

Indexes for arbitrary substring filters or `json_each` output will not be added: ordinary SQLite expression indexes cannot index the multiple rows emitted by a JSON table-valued expansion. Each proposed index must be justified by an exercised public query, not merely by a column appearing in a predicate.

### Treat list/count consolidation as an evidence-gated experiment

Artwork record listing and total counting currently repeat substantially the same filter predicates. A combined window-count query may reduce duplicate JSON and predicate work, but SQLite must still count all matches and may choose a worse sort plan. Implementation will compare the existing and combined forms on unfiltered, text, exact-artist, venue, and date-filter cases. The combined form will be retained only if results are identical and representative CPU/allocation evidence improves without a material regression in any supported case.

### Verify structure and measured effect separately

Deterministic tests will prove fragment read boundaries, co-author semantics, cache single-flight behaviour, invalidation, migration rollback, and unchanged HTTP/HTMX output. Query-plan inspection and production-shaped local profiles will then establish whether each structurally correct change improves the observed workload. Wall-clock thresholds will not be CI acceptance criteria because shared-runner timing is unstable; before-and-after commands, dataset cardinality, request concurrency, throughput, CPU samples, allocations, and retained heap will be reported together.

## Risks / Trade-offs

- [A reusable projection can become stale after a mutation] -> Invalidate from the owning record hooks and test save/delete plus in-flight invalidation races.
- [Coalescing loads can turn a slow loader into waiting callers] -> Keep projections bounded, preserve request admission ahead of handlers, share failures without caching them, and measure cold-load latency.
- [Caching complete projections raises retained heap] -> Store compact DTOs/sets rather than PocketBase records, measure forced-GC retained heap, and reject the approach if its bounded footprint is material.
- [New indexes increase database size and write cost] -> Add only plan-proven indexes, measure database growth, and provide a migration down path.
- [Results/full-view separation can drift semantically] -> Make the full view compose the same results workflow and retain shared filter parsing and canonical URL tests.
- [Synthetic benchmarks can hide production data distributions] -> Use deterministic tests for correctness and repeat local profiles against the production-shaped copied dataset for performance conclusions.

## Migration Plan

1. Establish repeatable baseline profiles and query plans for the agreed route/filter matrix.
2. Deploy workflow and bounded-cache changes without altering public contracts; caches begin empty and populate on demand.
3. Apply proven artist indexes through the normal PocketBase migration lifecycle.
4. Re-run the same profile matrix and compare CPU, allocation, throughput, and retained heap evidence.
5. Roll back application changes normally if semantics or resource use regress; roll back the index migration by removing only the newly named indexes. No persisted catalogue data is transformed.
