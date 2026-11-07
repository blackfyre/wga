# Spec Delta

## MODIFIED Requirements

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
