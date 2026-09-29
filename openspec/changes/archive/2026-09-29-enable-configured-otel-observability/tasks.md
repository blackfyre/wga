## 1. Configuration Contract

- [x] 1.1 Add optional `OTEL_EXPORTER_OTLP_ENDPOINT` parsing and validation to `internal/config`, expose it through server configuration, and verify focused config tests cover absent, local HTTP, Railway-private HTTP, external HTTPS, and malformed values.
- [x] 1.2 Update `.env.example` and durable configuration documentation to describe endpoint-driven enablement, private-network transport, and the absence of a separate enable flag; verify the repository documentation checks required by `docs/documentation-maintenance.md` pass.

## 2. Optional Telemetry Runtime

- [x] 2.1 Replace the development-gated trace adapter with an optional observability runtime that owns OTLP trace and metric providers, stable resource attributes, W3C propagation, bounded queues, and five-second shutdown; verify focused observability tests with in-memory exporters cover disabled and enabled runtimes without a live collector.
- [x] 2.2 Wire validated observability configuration through `cmd/wga` serving startup and shutdown while preserving non-serving commands and redacted failure logging; verify `cmd/wga` tests cover absent configuration, configured startup, and joined configuration failures.
- [x] 2.3 Preserve safe matched-route server spans and configured global-search child spans in every environment, removing environment-based assumptions; verify request and search tests assert propagation, stable attributes, error status, and absence of raw paths, query terms, identifiers, and arbitrary error text.

## 3. Cache and Repository Evidence

- [x] 3.1 Add allow-listed cache instrumentation for hit, miss, shared in-flight load, load failure, load duration, and invalidation without changing generation or single-flight semantics; verify focused cache tests cover concurrency, invalidation during load, loader failure, panic release, and disabled telemetry.
- [x] 3.2 Instrument the collection-holdings and published-artwork-author cold loaders with stable operation spans and duration/outcome metrics; verify repository and hook tests distinguish cold load, shared load, cache hit, mutation invalidation, and failure without exposing SQL, bound values, or record identifiers.

## 4. Public-Read Saturation Evidence

- [x] 4.1 Add bounded admission decision counters and current/configured capacity observations inside `internal/requestprotection`, using only stable profile, decision, and status dimensions; verify focused admission tests cover acquire, rate rejection, capacity rejection, release, observe mode, and the absence of client or route-value attributes.
- [x] 4.2 Verify the existing structured request-protection events and new metrics agree for controlled 429, 503, cancellation, and successful-release scenarios by running the focused handler/request-protection integration tests.

## 5. Railway Collector Pipeline

- [x] 5.1 Add a pinned Collector Contrib configuration and container/deployment files with private OTLP receivers, health check, memory limiter first, error/503/latency plus probabilistic tail sampling, unsampled metrics, batching, and bounded exporter queues/retries; verify the pinned collector accepts the configuration with placeholder backend settings.
- [x] 5.2 Document Railway service variables, private endpoint wiring, resource limits, backend credentials, UAT enablement, production rollout, health verification, and endpoint-removal rollback without creating a public collector domain; verify all examples use Railway reference variables and keep credentials out of WGA configuration.

## 6. Compliance and Deterministic Verification

- [x] 6.1 Reconcile Go dependencies and update reviewed licence, notices, and SBOM evidence for added OpenTelemetry metric/export components; verify dependency-maintenance and licence-generation checks pass with no unreviewed component.
- [x] 6.2 Run formatting, focused observability/config/cache/repository/request-protection tests, `go vet ./...`, `go test ./... -cover`, and the repository lint checks; correct only failures caused by this change.
- [x] 6.3 Exercise a local configured collector path to verify one controlled cache miss followed by a hit produces safe request/operation traces and aggregate cache metrics, then verify the same request with the endpoint absent emits no telemetry and still succeeds.
