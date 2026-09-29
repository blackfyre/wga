# Development OpenTelemetry Tracing Specification

## Purpose

Defines local Jaeger request tracing for development servers without enabling telemetry export in test, staging, or production.

## Requirements

### Requirement: Local Jaeger service startup
The development service task SHALL start Jaeger v2's all-in-one process in the background when its local health endpoint is unavailable, and SHALL wait for that endpoint before returning. The existing aggregate local-service task SHALL depend on the Jaeger startup task.

#### Scenario: Local services start without Jaeger already running
- **WHEN** a developer runs the aggregate local-service task while Jaeger is stopped
- **THEN** Jaeger starts in the background and the task returns only after its health endpoint responds

#### Scenario: Local services start with Jaeger already healthy
- **WHEN** a developer runs the aggregate local-service task while Jaeger is already healthy
- **THEN** it reuses the running process without starting a second instance

### Requirement: Unicode case-insensitive artist matching
Global search and the artist index SHALL match an artist's imported uppercase Unicode filing name when a visitor submits the equivalent normal-cased Unicode query. The query parameter SHALL retain its standard URL encoding and the correction SHALL NOT transliterate or alter the stored artist name.

#### Scenario: Imported uppercase Dürer filing name
- **WHEN** an artist is stored as `DÜRER, Albrecht` and a visitor searches `Dürer`
- **THEN** global search and the artist index include that artist in their results
