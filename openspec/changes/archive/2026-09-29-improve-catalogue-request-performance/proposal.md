## Why

Production traces show that Dual Mode repeatedly spends most of its request time loading stable reference data, while artwork-search facet construction consumes most of `/artworks` latency but remains too coarsely traced to identify the responsible query. WGA should remove the known repeated work and make the remaining facet cost actionable before attempting a speculative SQL rewrite.

## What Changes

- Reuse a bounded application-scoped Dual Mode reference projection across requests, with shared cold loading and mutation-driven invalidation.
- Split artwork-search facet tracing into stable option, venue, school-count, and form-count stages without exposing filters or catalogue identity.
- Preserve catalogue results, canonical URLs, progressive enhancement, cancellation, and existing cache and trace privacy guarantees.
- Explicitly defer facet SQL restructuring or lazy HTMX loading until the refined production evidence identifies the dominant substage.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: Extend the existing bounded-work requirement to reuse Dual Mode reference data safely and expose bounded diagnostic stages for expensive artwork facets.

## Impact

- Affects Dual Mode reference loading, catalogue mutation hooks, application cache telemetry, and artwork-search workflow tracing.
- Adds no public API, persistence migration, dependency, or visitor-visible interaction change.
- Requires regression coverage for cache coalescing, invalidation, cancellation, trace privacy, and unchanged catalogue rendering.
