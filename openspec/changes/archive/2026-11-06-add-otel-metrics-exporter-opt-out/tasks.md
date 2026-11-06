# Tasks

## 1. Configuration

- [x] 1.1 Parse `OTEL_METRICS_EXPORTER` in `internal/config` into the OpenTelemetry settings: empty or `otlp` enables metric export, `none` disables it (case-insensitive, trimmed), and any other value returns an error naming `OTEL_METRICS_EXPORTER`, including when no endpoint is set. Verify with table cases added to `go test ./internal/config -run '^TestServerOpenTelemetryConfiguration$'`.

## 2. Runtime

- [x] 2.1 In `internal/observability`, build the metric exporter and periodic reader only when metric export is enabled; otherwise install a no-op meter provider, shut down only the trace provider, and report metric export as disabled in the `observability.otel.enabled` startup event. Verify with a focused test in `internal/observability` that `none` yields a working tracer with a no-op meter provider, plus the existing `go test ./internal/observability`.
- [x] 2.2 Check that existing metric-instrument tests still pass with the change: `go test ./internal/utils ./internal/repositories ./internal/requestprotection ./internal/handlers`.

## 3. Documentation

- [x] 3.1 Document `OTEL_METRICS_EXPORTER` in `.env.example`, `docs/development-guide.md` (OpenTelemetry section) and `docs/observability-runbook.md` (use with trace-only receivers such as Jaeger, and how to roll back), following `docs/documentation-maintenance.md`. Verify with `bunx prettier --check` on the changed Markdown files.

## 4. Verification

- [x] 4.1 Run `go mod tidy`, `go vet ./...`, `go test ./... -cover` and `mise run check`, and confirm all pass.
