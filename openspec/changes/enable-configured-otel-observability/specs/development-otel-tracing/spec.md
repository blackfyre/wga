## REMOVED Requirements

### Requirement: Development-only request tracing

**Reason**: Telemetry enablement now follows explicit collector configuration so the same safe instrumentation can diagnose staging and production without being coupled to `WGA_ENV`.

**Migration**: Configure the local Jaeger OTLP endpoint explicitly for development; leave the endpoint absent wherever telemetry must remain disabled. Cross-environment telemetry behaviour is defined by `configured-otel-observability`.

### Requirement: Safe trace identity and request context

**Reason**: The safety and propagation contract now applies to every configured environment rather than development alone.

**Migration**: Use the equivalent safe service and request trace requirement in `configured-otel-observability`.

### Requirement: Global-search stage tracing

**Reason**: Global-search stage tracing now applies whenever telemetry is configured rather than only in development.

**Migration**: Use the expensive-operation tracing requirement in `configured-otel-observability`.

### Requirement: Bounded trace flush

**Reason**: Bounded shutdown now covers the configured traces and metrics pipeline in every environment.

**Migration**: Use the telemetry-failure and bounded-shutdown requirement in `configured-otel-observability`.
