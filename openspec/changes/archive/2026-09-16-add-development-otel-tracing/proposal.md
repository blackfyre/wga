## Why

Local request diagnosis currently relies on logs and Sentry. Developers need request traces in Jaeger while working locally without adding telemetry export to deployed environments.

## What Changes

- Add development-only OpenTelemetry request tracing that exports OTLP over insecure gRPC to the local Jaeger receiver at `localhost:4317`.
- Include Jaeger in the existing local-service startup task, waiting for its health endpoint before the task completes.
- Propagate inbound W3C trace context and record stable HTTP method, route, status, and error information.
- Attach stable service, release, and environment resource attributes, and flush spans with a bounded shutdown.
- Preserve the existing production, staging, and test runtime behaviour: they do not initialise or export OpenTelemetry traces.

## Capabilities

### New Capabilities

- `development-otel-tracing`: local Jaeger request tracing for development servers.

### Modified Capabilities

None.

## Impact

- Affected code: application startup, `internal/observability`, local Mise service tasks, Go dependency manifests, and licence metadata.
- Affected systems: a local Jaeger v2 process listening on OTLP gRPC port 4317.
- No public HTTP routes, responses, authentication, or production deployment configuration change.
