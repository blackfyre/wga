## Why

Railway resource and HTTP metrics show that expensive SQLite reads and capacity rejection can drive WGA into an overload cycle, but the deployed application cannot currently distinguish cache misses, shared cache loads, query stages, or capacity saturation. Existing OpenTelemetry support is restricted to development and a hard-coded local endpoint, so it cannot provide the production-safe evidence needed to diagnose these incidents.

## What Changes

- Enable OpenTelemetry only when an optional collector endpoint is configured, independently of `WGA_ENV`; an absent endpoint leaves telemetry disabled.
- Export vendor-neutral traces and metrics through OTLP to a configured collector while preserving bounded shutdown and non-disruptive request handling.
- Instrument stable, low-cardinality HTTP, cache, expensive repository-operation, and public-read capacity signals without recording raw SQL, visitor queries, record identifiers, client addresses, or payload content.
- Provide a Railway collector configuration that receives OTLP over the private network, limits memory, batches and retries telemetry, and forwards it to an operator-configured backend.
- Retain local Jaeger support by configuring its OTLP endpoint explicitly rather than enabling tracing merely because the runtime is in development.
- Define verification for disabled, locally enabled, and Railway-enabled telemetry paths without requiring an external collector in the default test suite.

Non-goals include caching arbitrary SQL results, replacing Sentry or structured application logs, selecting a permanent observability vendor, and treating telemetry as a substitute for query-plan optimisation.

## Capabilities

### New Capabilities

- `configured-otel-observability`: Optional endpoint-driven OTLP traces and metrics, safe cache/query/capacity instrumentation, and a bounded Railway collector pipeline.

### Modified Capabilities

- `development-otel-tracing`: Replace environment-driven enablement and the hard-coded local endpoint with explicit collector configuration while retaining safe request spans, local Jaeger compatibility, propagation, and bounded shutdown.

## Impact

- Affects `internal/config`, `internal/observability`, application startup wiring, cache utilities, expensive catalogue repositories, and public request protection.
- Updates local environment examples and development service wiring so local tracing is explicitly configured.
- Adds Railway collector deployment/configuration and operator-facing observability settings; no collector or backend is exposed publicly.
- May add OpenTelemetry metric SDK/exporter and collector components, requiring licence, notice, SBOM, and dependency verification.
- Does not change public HTTP response contracts, PocketBase persistence, or authoritative catalogue data.
