# Design

## Context

`internal/config` parses only `OTEL_EXPORTER_OTLP_ENDPOINT` into `config.OpenTelemetry`. `observability.ConfigureTracing` always builds an OTLP/gRPC trace exporter and an OTLP/gRPC metric exporter for that endpoint, with a 15-second periodic metric reader, and installs both providers globally. The exporters are built explicitly, so the SDK's own environment handling (`OTEL_METRICS_EXPORTER`, `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`) has no effect; `TestOTLPExportersIgnoreAmbientCredentials` deliberately keeps ambient `OTEL_*` settings out of the exporters.

## Goals / Non-Goals

**Goals:**

- Parse the opt-out in `internal/config`, as every deployment setting is.
- Keep metric instruments in feature code unchanged; they must work whether or not metric export is enabled.

**Non-Goals:**

- Adopting the SDK's `autoexport` package or general `OTEL_*` environment handling.

## Decisions

### Read `OTEL_METRICS_EXPORTER` through `internal/config`

`config.OpenTelemetry` gains a metrics-enabled flag, parsed from `OTEL_METRICS_EXPORTER` alongside the endpoint. Accepted values are empty, `otlp` and `none`, case-insensitive after trimming, matching the OpenTelemetry environment specification's case-insensitive enum handling. Other values produce a configuration error, including when no endpoint is set, so a typo fails at startup instead of silently keeping metric export on.

Alternative: use `go.opentelemetry.io/contrib/exporters/autoexport`. It would honour the variable, but it would also read other ambient `OTEL_*` settings and bypass the endpoint validation and credential isolation in `internal/config`. It also adds a dependency for a two-value switch.

Alternative: a WGA-specific flag such as `WGA_OTEL_METRICS`. Rejected because the standard name is what operators expect, and it documents itself.

### Use a no-op meter provider when metric export is disabled

`ConfigureTracing` creates the metric exporter and periodic reader only when metric export is enabled. Otherwise it uses `go.opentelemetry.io/otel/metric/noop` as the tracer's meter provider and as the global meter provider; the `metric` module is already a direct dependency. Shutdown then flushes only the trace provider. Instruments obtained from the no-op provider record nothing, so cache, repository and admission instrumentation need no change.

Alternative: keep an SDK meter provider with no reader. It would also export nothing, but it would still aggregate measurements for no purpose.

## Risks / Trade-offs

- [Operators set `none` and later wonder why metrics are missing] → The startup event reports metric export as disabled, and the runbook says when to use the setting.
- [Existing metric tests rely on the global meter provider] → Those tests install their own providers; the change only affects the runtime path when the setting is `none`.

## Migration Plan

No migration. Absent or `otlp` keeps current behaviour. For Railway `uat`, set `OTEL_METRICS_EXPORTER=none` on the `wga` service and redeploy; remove the variable to roll back.
