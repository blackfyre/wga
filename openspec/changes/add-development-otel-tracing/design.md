## Context

WGA uses PocketBase router middleware for request-scoped concerns. Jaeger v2's all-in-one local process accepts OTLP gRPC on `localhost:4317` and serves its UI on port 16686. The existing development configuration already installs its binary through Mise.

## Goals / Non-Goals

**Goals:**
- Trace every serving-runtime request only when `WGA_ENV=development`.
- Distinguish global-search lookup stages without recording a visitor's search term.
- Preserve inbound W3C trace context and use low-cardinality semantic HTTP attributes.
- Identify traces as WGA with the deployed release and environment.
- Flush queued spans within a fixed shutdown deadline.

**Non-Goals:**
- Production, staging, or test telemetry export.
- Metrics, browser tracing, automatic outbound HTTP/database instrumentation, or production Jaeger deployment configuration.
- Recording query strings, request bodies, cookies, client IP addresses, user data, or arbitrary error text.

## Decisions

### Export directly to local Jaeger through OTLP gRPC

The development runtime will send spans to Jaeger v2's default OTLP gRPC listener at `localhost:4317`, using insecure local transport. This keeps vendor-specific Jaeger client code out of WGA and aligns instrumentation with OpenTelemetry. The established Mise Jaeger binary remains the developer-owned local service; its default all-in-one mode provides the transient trace store and UI.

`mise run services:up` will depend on an idempotent Jaeger startup task. That task starts the all-in-one binary in the background only when its local health endpoint is unavailable, writes its output to `dist/jaeger.log`, and waits for the endpoint before returning. This follows the existing container service task without introducing a production deployment service.

### Instrument the PocketBase request middleware boundary

The tracing adapter will register one PocketBase router middleware. It extracts W3C Trace Context and baggage from the request headers, creates a server span, places its context back onto the request, and ends the span after the handler returns. It records only the request method, router pattern, response status, and returned error signal. Router patterns avoid high-cardinality raw paths and no request content is added as an attribute.

The global search workflow will add child spans for artist lookup, artwork lookup, and result-author hydration. These spans name stable workflow stages only; the search term and result content remain absent. This identifies whether the request time is spent in filter/count queries or per-result author reads without changing query behaviour.

### Match uppercase Unicode filing names from normal visitor input

SQLite's built-in case-insensitive text matching covers ASCII only. The imported catalogue preserves uppercase filing names such as `DÜRER, Albrecht`, so a conventional title-case query such as `Dürer` fails the artist predicate even though its URL is decoded correctly. The global search and artist index will bind a Unicode uppercase variant alongside the visitor term for their artist-filing-name predicates. This is a narrow compatibility correction for the imported filing-name convention; it does not change stored data or introduce locale-specific transliteration.

### Limit tracing to development at application startup

`cmd/wga` will construct and register the tracer only for serving commands. The observability package will return a disabled tracer outside development, so production, staging, and tests neither construct the OTLP exporter nor install middleware. Shutdown uses a five-second context timeout and logs a bounded flush failure without altering PocketBase's shutdown result.

## Operational Decomposition

| Workstream | Area of operations | Coordination and sequencing |
|---|---|---|
| Tracing adapter | `internal/observability` and its tests | Defines the OTLP resource, propagation, request span, and bounded shutdown contract. |
| Application wiring | `cmd/wga/main.go` | Registers the adapter after request IDs and flushes it wherever Sentry is currently flushed. |
| Local Jaeger service | `mise.toml` | Starts Jaeger before the existing `services:up` task completes. |
| Search tracing | `internal/handlers/search` and focused tests | Adds safe child spans below the development request trace. |
| Unicode artist matching | global search and artist-index repository tests | Applies a Unicode uppercase query variant wherever a visitor searches filing names. |
| Dependency evidence | `go.mod`, `go.sum`, `internal/licences/manifest.json`, generated notices/SBOM | Follows the final Go module graph after the adapter compiles. |
| Verification | focused observability test, static checks, and OpenSpec validation | Runs after all implementation changes. |

## Risks / Trade-offs

- [Jaeger is not running] → the OTLP batcher retries asynchronously; WGA continues, and its bounded shutdown prevents indefinite exit delay.
- [Request values leak into traces] → attach only method, route pattern, status, and SDK-generated error signal; do not record raw URLs or error messages.
- [Trace context is absent or malformed] → the propagator creates a new root trace without changing request handling.

## Migration Plan

1. Run the installed `jaeger` binary locally; its default all-in-one configuration exposes Jaeger at `http://localhost:16686` and OTLP gRPC at `localhost:4317`.
2. Start WGA with `WGA_ENV=development` and make a request.
3. Inspect the `wga` service traces in Jaeger.
4. Roll back by removing the tracing dependency and startup wiring; no stored application data is changed.

## Open Questions

None.
