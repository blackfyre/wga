# Proposal

## Why

`OTEL_EXPORTER_OTLP_ENDPOINT` enables trace and metric export together. When WGA points at a trace-only OTLP receiver, such as the Jaeger service in the Railway `uat` environment, every metric export is rejected with `Unimplemented … MetricsService`, and the OpenTelemetry SDK logs `failed to upload metrics` every 15 seconds. Operators need to keep traces while turning metric export off, without deploying a collector only to discard metrics.

## What Changes

- WGA honours the standard OpenTelemetry setting `OTEL_METRICS_EXPORTER`. Unset or `otlp` keeps today's behaviour; `none` disables metric export while trace export and request tracing continue.
- Any other `OTEL_METRICS_EXPORTER` value fails server configuration with an actionable validation error, consistent with the existing endpoint validation.
- With metrics disabled, the application's metric instruments record nothing and no metric export is attempted.
- The startup log reports whether metric export is enabled.
- Documentation (`.env.example`, development guide, observability runbook) describes the setting and its intended use with trace-only receivers.

Non-goals:

- No other `OTEL_METRICS_EXPORTER` values (`prometheus`, `console`) and no separate metrics endpoint.
- No equivalent opt-out for traces (`OTEL_TRACES_EXPORTER`).
- No change to the collector configuration or to the default behaviour when the setting is absent.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `configured-otel-observability`: the "Collector configuration controls telemetry" requirement gains an operator opt-out for metric export, and its validation covers the new setting.

## Impact

- `internal/config`: parse and validate `OTEL_METRICS_EXPORTER` as part of the OpenTelemetry settings.
- `internal/observability/tracing.go`: build the metric exporter and periodic reader only when metric export is enabled; otherwise use a no-op meter provider.
- Tests in `internal/config` and `internal/observability`.
- `.env.example`, `docs/development-guide.md`, `docs/observability-runbook.md`.
- Operations: the Railway `uat` WGA service can set `OTEL_METRICS_EXPORTER=none` to stop the log noise while it exports traces to Jaeger.
