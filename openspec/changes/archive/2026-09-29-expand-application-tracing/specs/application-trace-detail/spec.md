## Purpose

Defines bounded application-level trace detail that explains material catalogue work without exposing visitor input, catalogue identity, or database content.

## ADDED Requirements

### Requirement: Material catalogue workflows expose stable trace stages

Configured telemetry SHALL emit child spans for selected material workflow stages in ordinary catalogue requests. Stage names SHALL come from a finite allow-list and SHALL describe the kind of work rather than request or record content.

#### Scenario: Dual-mode page is assembled

- **WHEN** a configured runtime assembles a dual-mode response
- **THEN** its request trace distinguishes reference loading, pane construction, and response rendering with stable child spans

#### Scenario: Artwork search is assembled

- **WHEN** a configured runtime assembles an artwork-search response
- **THEN** its request trace distinguishes material result and facet work with stable child spans

### Requirement: Material repository operations are distinguishable

Configured telemetry SHALL emit child spans with bounded names and success or failure outcomes for allow-listed repository operations that perform material catalogue queries. It SHALL NOT emit a child span for every record, relation hydration, formatting helper, or trivial lookup.

#### Scenario: Artist index performs material queries

- **WHEN** an artist-index workflow counts and loads a bounded page of artists
- **THEN** the request trace distinguishes the allow-listed count and list operations without identifying the filter or returned artists

#### Scenario: Repository operation fails

- **WHEN** an allow-listed repository operation returns an error
- **THEN** its span records a failure outcome without recording arbitrary error text

### Requirement: Cache outcomes enrich the active request span

An instrumented cache request made with an active recording span SHALL add one bounded cache event to that span identifying only the allow-listed cache name and the outcome `hit`, `miss`, `shared`, or `failure`. Cache access SHALL NOT create a child span solely for a hit or shared result.

#### Scenario: Request reuses a cached projection

- **WHEN** a traced request receives an existing instrumented projection
- **THEN** the active trace records a cache event with the stable cache name and `hit` outcome without recording the cache key

#### Scenario: Cold cache performs authoritative work

- **WHEN** a traced request misses an instrumented cache and executes an allow-listed authoritative load
- **THEN** the trace contains the bounded cache outcome event and the authoritative operation span

### Requirement: Expanded trace detail remains privacy-safe and bounded

Application child spans and events SHALL NOT contain raw SQL, bound values, search terms, query strings, URLs, record identifiers, result content, visitor data, cache keys, or arbitrary error text. Trace detail SHALL remain inactive when configured telemetry is disabled and SHALL NOT change request results when tracing fails.

#### Scenario: Visitor-controlled filters are present

- **WHEN** an instrumented catalogue request contains visitor-controlled filters or record paths
- **THEN** its child spans and events contain only allow-listed operation, stage, cache, and outcome values

#### Scenario: Telemetry is disabled

- **WHEN** the same catalogue workflow runs without a configured collector endpoint
- **THEN** it preserves its existing result and does not perform externally observable telemetry work
