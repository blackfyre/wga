# configured-otel-observability Specification

## Purpose

Defines optional, collector-configured OpenTelemetry traces and metrics that diagnose WGA request, cache, database-operation, and admission behaviour without exposing visitor or catalogue data.

## Requirements

### Requirement: Collector configuration controls telemetry

The serving runtime SHALL initialise OpenTelemetry export only when a valid collector endpoint is configured. Enablement SHALL NOT depend on the deployment environment, and the runtime SHALL remain telemetry-free when the endpoint is absent.

When a collector endpoint is configured, the runtime SHALL export metrics unless `OTEL_METRICS_EXPORTER` is set to `none`. With metric export disabled, the runtime SHALL continue to export traces and register telemetry middleware, and SHALL NOT attempt any metric export. `OTEL_METRICS_EXPORTER` SHALL accept only an absent or empty value, `otlp`, or `none`, compared case-insensitively after trimming surrounding whitespace; any other value SHALL fail server configuration whether or not an endpoint is configured. The startup event that reports enabled telemetry SHALL state whether metric export is enabled.

#### Scenario: No collector endpoint is configured

- **WHEN** WGA starts without an OpenTelemetry collector endpoint
- **THEN** it does not initialise OTLP exporters, register telemetry middleware, or emit application telemetry

#### Scenario: Collector endpoint is configured outside development

- **WHEN** WGA starts in staging or production with a valid collector endpoint
- **THEN** it initialises the configured OTLP telemetry pipeline

#### Scenario: Collector endpoint is malformed

- **WHEN** WGA starts with an OpenTelemetry collector endpoint that does not satisfy the supported endpoint contract
- **THEN** server configuration fails with an actionable validation error before serving requests

#### Scenario: Metric export is left at its default

- **WHEN** WGA starts with a valid collector endpoint and `OTEL_METRICS_EXPORTER` is absent, empty, or `otlp`
- **THEN** it exports both traces and metrics to the configured endpoint

#### Scenario: Metric export is disabled

- **WHEN** WGA starts with a valid collector endpoint and `OTEL_METRICS_EXPORTER=none`
- **THEN** it exports traces to the configured endpoint, sends no metric export requests, and its startup event reports metric export as disabled

#### Scenario: Metric exporter value is unsupported

- **WHEN** WGA starts with `OTEL_METRICS_EXPORTER` set to a value other than empty, `otlp`, or `none`
- **THEN** server configuration fails with an actionable validation error that names `OTEL_METRICS_EXPORTER` before serving requests

### Requirement: Safe service and request traces

Configured telemetry SHALL identify the service name, release version, and deployment environment, SHALL extract valid W3C Trace Context and baggage, and SHALL describe inbound requests with stable HTTP method, matched route pattern, response status, and error status only. It SHALL NOT add raw URLs, query strings, headers, cookies, bodies, client addresses, user identifiers, catalogue identifiers, or arbitrary error text.

#### Scenario: Request continues an existing trace

- **WHEN** a request contains valid W3C trace context and telemetry is configured
- **THEN** the server span continues that trace and carries only the permitted low-cardinality request attributes

#### Scenario: Request contains sensitive values

- **WHEN** a request path or query contains visitor-controlled or catalogue-specific values
- **THEN** emitted telemetry identifies the matched route pattern without recording those values

### Requirement: Expensive operations are distinguishable

Configured telemetry SHALL expose durations and outcomes for the stable catalogue operations known to perform material database or projection work. Operation names SHALL be bounded and SHALL NOT contain raw SQL, bound values, search terms, record identifiers, result content, or arbitrary error text.

#### Scenario: Collection holdings projection loads

- **WHEN** the collection-holdings projection executes its authoritative database operation
- **THEN** telemetry records the stable operation name, duration, and success or failure outcome without recording query content

#### Scenario: Artist availability projection loads

- **WHEN** the published-artwork-author projection executes its authoritative database operation
- **THEN** telemetry records the stable operation name, duration, and success or failure outcome without recording author identifiers

#### Scenario: Global search executes

- **WHEN** a configured runtime handles global search
- **THEN** child spans distinguish artist lookup, artwork lookup, and author hydration without recording the submitted term or returned records

### Requirement: Cache behaviour is measurable

Configured telemetry SHALL report bounded cache request outcomes for hit, miss, shared in-flight load, and load failure; SHALL report cache-load duration and invalidation count; and SHALL identify caches only by an allow-listed stable name.

#### Scenario: Cached projection is reused

- **WHEN** a caller receives an existing projection from the application cache
- **THEN** the cache request metric records one hit for the stable cache name without creating a database-operation span

#### Scenario: Concurrent callers share a cold load

- **WHEN** concurrent callers request the same uncached projection while one loader is active
- **THEN** telemetry distinguishes the single miss and load from the callers that shared the in-flight result

#### Scenario: Projection is invalidated

- **WHEN** an authoritative mutation invalidates an instrumented projection
- **THEN** telemetry increments the invalidation count for that stable cache name

### Requirement: Public-read saturation is measurable

Configured telemetry SHALL report bounded admission counts by stable route profile, decision, and HTTP status, together with current and configured concurrent-work utilisation. It SHALL NOT report raw or derived client identities or requested record paths.

#### Scenario: Capacity rejects a request

- **WHEN** public-read admission rejects a request because all concurrent-work slots are occupied
- **THEN** telemetry records one capacity rejection and bounded utilisation values for the route profile

#### Scenario: Protected work is admitted and released

- **WHEN** a protected request acquires and later releases a capacity lease
- **THEN** the reported current utilisation reflects both transitions without identifying the requester

### Requirement: Telemetry failure does not become request failure

After successful startup configuration, collector unavailability, export retries, dropped telemetry, or sampling SHALL NOT change application responses or block request processing. Shutdown SHALL attempt to flush configured telemetry within a deadline no longer than five seconds and SHALL log a bounded, redacted failure when flushing does not complete.

#### Scenario: Collector becomes unavailable

- **WHEN** the configured collector cannot accept telemetry while WGA is serving requests
- **THEN** WGA continues serving requests while the telemetry pipeline applies its bounded retry and drop behaviour

#### Scenario: Collector is unavailable during shutdown

- **WHEN** WGA stops while configured telemetry cannot be exported
- **THEN** the process exits after the bounded flush attempt and records a redacted failure event

### Requirement: Railway collector is private and bounded

The Railway deployment SHALL provide a collector that accepts application OTLP traffic only through Railway private networking, applies memory limiting before filtering or sampling, batches export, retries transient backend failures within bounded queues, exposes an operational health check, and forwards telemetry to an operator-configured backend without embedding backend credentials in WGA.

#### Scenario: WGA exports inside Railway

- **WHEN** Railway deploys WGA with the collector's private endpoint configured
- **THEN** WGA sends OTLP traffic over Railway private networking without requiring a public collector domain

#### Scenario: Collector approaches its memory limit

- **WHEN** buffered telemetry approaches the collector's configured memory boundary
- **THEN** the collector sheds telemetry according to its bounded policy instead of consuming unbounded memory

#### Scenario: Backend credentials are required

- **WHEN** the collector exports to an authenticated telemetry backend
- **THEN** credentials are supplied to the collector service and are not present in WGA configuration or telemetry attributes

### Requirement: Trace retention is bounded without losing incident evidence

The deployed collector SHALL allow healthy successful traces to be sampled while retaining traces that contain server errors, capacity rejections, or requests exceeding the configured slow-request threshold. Aggregate metrics SHALL remain independent of trace sampling.

#### Scenario: Healthy request volume is high

- **WHEN** successful requests exceed the configured trace-retention rate
- **THEN** the collector may discard healthy traces without reducing the corresponding aggregate metric counts

#### Scenario: Request is slow or rejected

- **WHEN** a trace contains a configured slow request, server error, or capacity rejection
- **THEN** the collector retains that trace for backend export
