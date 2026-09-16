## ADDED Requirements

### Requirement: Development-only request tracing
The serving runtime SHALL initialise OpenTelemetry tracing only when the configured WGA environment is `development`. It SHALL export spans through OTLP gRPC to the local Jaeger listener at `localhost:4317`, and it SHALL not initialise an OTLP exporter or register tracing middleware in test, staging, or production environments.

#### Scenario: Development server receives a request
- **WHEN** WGA serves a request with `WGA_ENV=development` and Jaeger accepts OTLP traffic on `localhost:4317`
- **THEN** Jaeger receives a server span for the matched request

#### Scenario: Non-development server starts
- **WHEN** WGA starts with test, staging, or production environment configuration
- **THEN** it does not create an OTLP exporter or emit request traces

### Requirement: Local Jaeger service startup
The development service task SHALL start Jaeger v2's all-in-one process in the background when its local health endpoint is unavailable, and SHALL wait for that endpoint before returning. The existing aggregate local-service task SHALL depend on the Jaeger startup task.

#### Scenario: Local services start without Jaeger already running
- **WHEN** a developer runs the aggregate local-service task while Jaeger is stopped
- **THEN** Jaeger starts in the background and the task returns only after its health endpoint responds

#### Scenario: Local services start with Jaeger already healthy
- **WHEN** a developer runs the aggregate local-service task while Jaeger is already healthy
- **THEN** it reuses the running process without starting a second instance

### Requirement: Safe trace identity and request context
Development server spans SHALL declare `service.name`, `service.version`, and `deployment.environment.name` resource attributes. They SHALL extract and continue valid W3C Trace Context and baggage from inbound requests. Each request span SHALL record only the HTTP request method, matched router pattern, response status, and error status needed to diagnose the request; it SHALL NOT add raw URLs, query strings, headers, cookies, bodies, client IP addresses, user identifiers, or arbitrary error text.

#### Scenario: Request continues an existing trace
- **WHEN** a development request contains a valid `traceparent` header
- **THEN** its server span uses the supplied trace as its parent

#### Scenario: Request includes sensitive URL data
- **WHEN** a development request path or query contains user-controlled or sensitive values
- **THEN** the emitted span identifies only its matched route pattern and contains none of those values

### Requirement: Global-search stage tracing
For a development global-search request, the runtime SHALL emit child spans for artist lookup, artwork lookup, and artwork-author hydration. These spans SHALL use stable stage names and SHALL NOT include the raw search query, result titles, artist names, or record identifiers as attributes.

#### Scenario: Unicode search request
- **WHEN** a development visitor requests `/search?q=D%C3%BCrer`
- **THEN** Jaeger shows the request span and its global-search stage spans without exposing `Dürer` in span attributes

### Requirement: Unicode case-insensitive artist matching
Global search and the artist index SHALL match an artist's imported uppercase Unicode filing name when a visitor submits the equivalent normal-cased Unicode query. The query parameter SHALL retain its standard URL encoding and the correction SHALL NOT transliterate or alter the stored artist name.

#### Scenario: Imported uppercase Dürer filing name
- **WHEN** an artist is stored as `DÜRER, Albrecht` and a visitor searches `Dürer`
- **THEN** global search and the artist index include that artist in their results

### Requirement: Bounded trace flush
The development serving runtime SHALL attempt to flush queued spans after PocketBase stops, with a deadline no longer than five seconds. A flush failure SHALL be logged without blocking process exit indefinitely.

#### Scenario: Jaeger is unavailable during shutdown
- **WHEN** the development server stops while Jaeger is unavailable
- **THEN** the server exits after the bounded trace flush attempt
