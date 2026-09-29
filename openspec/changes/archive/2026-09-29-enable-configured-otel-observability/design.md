## Context

See `proposal.md` for motivation. WGA currently creates a development-only trace provider, exports OTLP gRPC to hard-coded `localhost:4317`, registers one PocketBase request interceptor, and flushes spans within five seconds. The current cache helper already coalesces concurrent misses and the two measured catalogue-wide projections are application-scoped caches with mutation invalidation, but neither cache behaviour nor their cold database loads are observable. Public-read admission exposes bounded utilisation internally and emits structured decision logs, but no aggregate application metrics.

The existing `development-otel-tracing` specification explicitly prohibits staging and production export. This change moves the cross-environment safety, request tracing, search-stage tracing, and shutdown contracts into the new `configured-otel-observability` capability while retaining the local Jaeger service and unrelated Unicode-search behaviour in the existing capability.

## Goals / Non-Goals

**Goals:**

- Make collector endpoint presence the sole telemetry enablement switch in every environment.
- Produce enough traces and metrics to distinguish slow repository work, cache reuse or cold loads, and public-read saturation.
- Keep instrumentation vendor-neutral, low-cardinality, redaction-safe, bounded under collector failure, and testable without a live collector.
- Supply a single-replica Railway collector pipeline over private networking with bounded memory and backend-neutral export.

**Non-Goals:**

- Cache arbitrary SQL results or alter the authoritative cache invalidation model.
- Capture raw SQL, query parameters, visitor input, catalogue identifiers, client identity, request bodies, or unrestricted errors.
- Replace Sentry, structured JSON logs, Railway platform metrics, or query-plan optimisation.
- Select, provision, or define retention for a permanent telemetry backend.

## Decisions

### Use the standard OTLP endpoint as the capability switch

`internal/config` will parse the optional `OTEL_EXPORTER_OTLP_ENDPOINT` setting into server-owned observability configuration. An absent value produces a disabled telemetry runtime regardless of `WGA_ENV`; a valid value enables traces and metrics in any environment; a malformed or unsupported endpoint joins normal server configuration errors before PocketBase starts.

The endpoint contract will accept an explicit `http` or `https` URL and preserve the scheme's transport security choice. Railway will use `http` with the collector's private domain because Railway encrypts private-network traffic; external collector endpoints require `https`. Local Jaeger remains available through an explicitly configured localhost endpoint. No separate enable flag is introduced because it could conflict with endpoint presence.

Alternative considered: retain environment gating and add production exceptions. Rejected because environment identity does not express whether a collector exists and creates combinations where a configured endpoint is silently ignored.

### Own one optional telemetry runtime

The observability package will replace the trace-only adapter with one optional runtime that owns the trace provider, meter provider, propagator, request interceptor, and bounded shutdown. Startup passes validated configuration from `internal/config`; handlers and repositories receive or resolve only the narrow instrumentation contracts they need. Disabled instruments remain no-op and do not register request middleware or exporters.

Both signals use OTLP over the same configured endpoint and stable resources: `service.name=wga`, the build version, and deployment environment. SDK batch queues and export timeouts are explicitly bounded so collector failure cannot create unbounded memory growth or block request processing. Shutdown shares the existing five-second maximum across both providers and reports redacted error types through repository logging helpers.

Alternative considered: let OpenTelemetry libraries read process environment directly. Rejected because deployment settings are owned and validated by `internal/config`, and direct environment reads would bypass the application's configuration contract and tests.

### Instrument semantic operations, not SQL statements

Request middleware continues to create one server span using matched route patterns. Manual child spans and duration instruments are added only around stable, material operations: collection-holdings load, published-artwork-author load, and the existing global-search stages. Attribute values are closed sets such as operation name and outcome; SQL text, bound values, row identifiers, result content, and arbitrary errors are excluded.

The two cold projection loaders receive spans at the repository boundary rather than through blanket database-driver instrumentation. This exposes the work that matters without producing a span for every PocketBase query or risking query-statement cardinality and data leakage.

### Measure cache state transitions at the shared loader boundary

The shared cache loader will accept a stable telemetry identity selected from an application-owned allow-list for instrumented caches. It records hit, miss, shared in-flight load, failure, load duration, and invalidation while preserving the existing generation and single-flight semantics. Uninstrumented cache uses remain no-op; internal storage keys are not automatically promoted to telemetry dimensions.

This measures whether a slow request was a genuine cold load, whether concurrent callers coalesced successfully, and whether repeated invalidation explains load frequency. It deliberately does not cache complete request responses or introduce TTLs.

### Measure admission decisions and utilisation without client dimensions

The public-request protection package will emit counters for stable profile, decision, and response-status combinations and observable current/configured capacity values at acquire, reject, and release transitions. Instrumentation remains inside the owning admission package, while request handlers continue to map decisions to HTTP responses.

No raw address, private derived identity, requested path, or record slug becomes an attribute. Existing structured decision events remain authoritative detailed logs; metrics provide aggregation and alerts.

### Run one private, bounded collector on Railway

Repository-owned collector configuration will use the Collector Contrib distribution because tail sampling and operational extensions are required. One Railway service will:

1. receive OTLP gRPC and HTTP on private interfaces;
2. apply `memory_limiter` first;
3. apply trace filtering and tail-sampling policies that retain errors, 503s, and requests above a configurable latency threshold while probabilistically sampling ordinary successes;
4. batch traces and metrics after filtering;
5. export through bounded sending queues and retries to an operator-configured OTLP backend; and
6. expose a local health-check endpoint without a public Railway domain.

A single collector replica keeps every trace's spans at one tail-sampling decision point. Metrics bypass trace sampling. Backend endpoints, headers, and credentials are collector variables and never WGA settings or committed files.

Alternative considered: export WGA directly to the final backend. Rejected because it couples application configuration to a vendor and removes central sampling, retry, filtering, and future fan-out.

### Verify instrumentation with in-memory exporters and configuration inspection

Focused tests will inject in-memory trace and metric readers to assert enablement, safe attributes, cache outcome accounting, loader coalescing, operation durations, admission transitions, and bounded shutdown. Default Go tests will not require Jaeger, a collector, Railway, or a vendor backend. Collector configuration will be validated by starting the pinned collector image or its configuration-check command; Railway verification will send a controlled request and confirm both a trace and the corresponding aggregate metrics in the configured backend.

## Risks / Trade-offs

- [Instrumentation adds latency or allocation] -> Keep operation coverage narrow, use closed-set attributes, benchmark hot cache and admission paths, and leave disabled instruments as no-op.
- [A telemetry outage consumes application memory] -> Bound SDK queues, export timeouts, collector queues, and collector memory; allow telemetry loss rather than request failure.
- [Tail sampling drops useful healthy traces] -> Preserve all errors, 503s, and configured slow traces while retaining a small probabilistic healthy sample; keep metrics unsampled.
- [Cache telemetry changes single-flight correctness] -> Preserve generation locking and panic/error semantics, and extend the existing concurrency and invalidation tests with telemetry assertions.
- [Collector becomes another operational dependency] -> Give it an independent health check and resource limits; WGA remains available when it is unhealthy and can disable export by removing one endpoint setting.
- [Telemetry leaks sensitive or high-cardinality data] -> Use allow-listed operation/cache/profile names, test forbidden attributes, and keep backend credentials solely on the collector.
- [Cold loads remain expensive] -> Treat telemetry as diagnosis evidence only; continue query/index/read-model optimisation independently.

## Migration Plan

1. Add and validate the collector configuration against a non-production OTLP backend, then deploy it as a private Railway service with explicit CPU and memory limits.
2. Deploy WGA's configuration and instrumentation changes with `OTEL_EXPORTER_OTLP_ENDPOINT` absent; behaviour remains telemetry-disabled in every environment.
3. Configure the collector's private OTLP endpoint in UAT, issue controlled cache-hit, cold-load, and capacity scenarios, and confirm redaction, trace retention, and unsampled metrics.
4. Tune healthy-trace sampling and the slow-request threshold from UAT volume before enabling the endpoint in production.
5. Enable production export by setting only the validated collector endpoint on WGA; keep vendor credentials on the collector.
6. Roll back application emission by removing the WGA endpoint variable. The collector can then be rolled back independently without changing application responses or data.
