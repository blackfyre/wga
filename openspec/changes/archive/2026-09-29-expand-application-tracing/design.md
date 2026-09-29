## Context

Configured tracing currently provides one server span for every request, dedicated global-search stage spans, and two cold projection-load spans. Cache reuse is visible only through metrics. The dual-mode and artwork-search workflows already pass request contexts through their material work, while repository operations are identified by a small typed allow-list.

The existing configured-observability contract forbids request content, record identity, raw SQL, and arbitrary errors in telemetry. Collector-side tail sampling and exporter failure isolation remain unchanged.

## Goals / Non-Goals

**Goals:**

- Make representative dual-mode, artwork-search, and artist-index traces explain their material work.
- Preserve stable, finite span names and attributes suitable for tail sampling and aggregation.
- Surface cache outcomes inside traces without producing a child span for every cache access.
- Keep instrumentation testable with in-memory OpenTelemetry exporters.

**Non-Goals:**

- Automatic instrumentation of `database/sql`, PocketBase internals, templates, or every repository method.
- Recording SQL statements, filter values, route parameters, record identifiers, result sizes, or errors as text.
- Replacing the existing cache, operation-duration, or admission metrics.

## Decisions

### Use typed allow-lists for workflow and repository spans

Extend the observability package's existing typed operation approach so span names and permitted attributes are selected by code, not supplied as free-form request data. Instrument only material boundaries whose duration can guide diagnosis: dual-mode reference/pane/render stages, artwork-search result/facet stages, and artist-index count/list operations.

This keeps feature packages responsible for deciding what constitutes material work while the shared observability package owns naming, outcome, and privacy rules. Blanket SQL instrumentation was rejected because it would create high volume and risk exposing statement or bound-value content.

### Attach cache outcomes as events to the current span

The instrumented cache helper will add one event to `trace.SpanFromContext(ctx)` after resolving each request outcome. Events will contain only the existing stable cache name and bounded outcome. Existing aggregate cache metrics remain authoritative for rates and counts.

Creating cache child spans was rejected because cache hits are frequent and usually too short to justify the resulting span volume. Request-span attributes were also rejected because one request can access the same or multiple caches more than once, while events preserve occurrence order without an unbounded attribute scheme.

### Keep workflow spans coarse and repository spans selective

Workflow spans group meaningful sequential stages; repository spans identify only operations likely to dominate those stages. Per-record hydration loops, URL construction, parsing, formatting, and template helpers remain inside their parent stage.

The first coverage set is intentionally limited to dual mode, artwork search, and the shared artist index because these are ordinary catalogue paths with multi-stage work and existing request-context propagation. Further routes require a later evidence-based extension to the allow-list.

## Risks / Trade-offs

- [Risk] Added spans increase export volume for busy catalogue routes. → Keep the allow-list small, avoid loop-level spans, and retain collector tail sampling.
- [Risk] Nested workflow and repository spans can duplicate timing boundaries. → Use workflow spans for coarse stage ownership and repository spans only where the query is independently actionable.
- [Risk] A context-free compatibility wrapper can create unrelated root spans. → Instrument only context-aware request paths; leave background-context wrappers behaviourally unchanged unless they are deliberately given a parent context.
- [Risk] Cache events may be absent when no span records the request. → Preserve metrics as the complete aggregate signal; events are trace enrichment only.

## Migration Plan

Deploy as an application-only change with no configuration or persistence migration. Verify representative trace shapes locally and in UAT. Roll back the application release to remove the added spans and events; existing exporters, metrics, and collector configuration remain compatible.
