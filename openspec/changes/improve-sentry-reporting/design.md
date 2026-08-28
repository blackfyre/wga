## Context

See proposal.md for motivation. The request-ID middleware is registered before Sentry and stores a server-generated ID on both the request event and request context. The Sentry monitor currently captures returned server errors globally, but represents an HTTP 5xx written by a handler as a generic status-only error. The shared server-fault renderer does not retain an originating error for the monitor. Browser configuration contains only a DSN and environment; Bun generates source maps but the release pipeline does not upload them to Sentry.

## Goals / Non-Goals

**Goals:**
- Correlate every captured unexpected server failure with an allowlisted request diagnosis and one structured log event.
- Preserve a sanitised causal diagnosis for server-fault responses where the source has one.
- Give the server and browser a shared, immutable release value and make matching browser source maps available only to Sentry.
- Prove the privacy boundary and release artefact behaviour automatically.

**Non-Goals:**
- Adding performance tracing, session replay, profiling, or metrics.
- Changing routes, response bodies, status codes, authentication, or normal request logging.
- Fixing the stale browser-asset failure or adding broad browser-noise suppression.
- Exposing source maps to browsers or adding a runtime Sentry upload endpoint.

## Decisions

### Use a request-scoped failure record as the server-fault hand-off

The shared server-fault response boundary will accept an optional causal failure record and attach it only to the in-flight request event. A small neutral request-failure package will own this request-event contract so that both the response adapter and observability adapter can use it without creating an import cycle. The monitor will read that record after the handler returns and use it in preference to constructing a generic status exception. Existing 5xx responses and propagation remain unchanged.

The record will contain a stable, caller-controlled failure category and the originating error. Reporting will serialise only the category, error type, and sanitised exception information; it will not serialise an arbitrary error message or request data. Callers without a causal error will leave the record absent, producing the explicit status-only fallback required by the specification.

This keeps business workflows unaware of Sentry while letting their HTTP adapters preserve causal information at the point it would otherwise be discarded. Capturing the raw error directly at every handler was rejected because it would couple feature handlers to the monitoring vendor and would make privacy enforcement inconsistent.

### Capture from an isolated Sentry scope with an allowlisted diagnosis

For each report, the monitor will create an isolated Sentry scope that includes only:

- server-generated request ID;
- HTTP method;
- normalised router pattern, with a constant fallback for unmatched routes;
- response status;
- stable failure category and error type; and
- a fingerprint built from stable failure dimensions, never a raw path or request value.

The monitor will also emit one `observability.request.failed` structured log with those fields and the same request ID. It will not attach the request object to Sentry. This satisfies the correlation rule in ADR 0004 without allowing concurrent requests to share scope data.

Using raw URL paths was rejected because dynamic record IDs would create high-cardinality tags and issue groups. Relying solely on existing handler logs was rejected because not every written 500 produces a request-scoped, consistently named failure log.

### Treat privacy as a reporting allowlist and sanitisation boundary

The reporting adapter will scrub exception text before capture and use safe failure categories and error types for diagnosis. The same fail-closed sanitiser will process returned errors, recorded causes, wrapped errors, and panic values. It will not add query strings, fragments, headers, cookies, bodies, client IPs, user identifiers, or caller-supplied text to Sentry tags, contexts, fingerprints, or the matching log. Browser console breadcrumbs will discard both their message and argument data; malformed URLs will resolve to a safe placeholder rather than their original value.

Tests will exercise returned, wrapped, recorded, and panic failures containing representative sensitive values and assert their absence from both captured event data and logs. Browser tests will cover console messages as well as arguments and malformed URLs.

### Embed one explicit release value in the image for both runtimes

The Docker build will derive its immutable release from `git rev-parse HEAD`. The Go linker will embed it in `internal/buildinfo.Version`; the server monitor and browser layout will use that embedded value. The browser layout will expose only the release alongside its already-public DSN and environment.

Embedding the release in the image rather than reading a runtime setting guarantees that the server event and rendered browser metadata refer to the exact artefact Railway runs. A release is operational metadata, not a secret.

### Upload source maps during release assembly, then exclude them from static assets

The frontend release step will generate source maps in a clean staging directory outside the public asset tree and copy only non-map assets into the embedded public tree. The final image will retain the staged maps in a non-public path. At container start, an entrypoint will use the same embedded release identifier and a sealed Railway runtime token to upload maps to the browser project before executing WGA. The authenticated upload token is not added to application configuration, Docker build arguments, command arguments, or browser markup.

The entrypoint must fail before WGA starts if an enabled browser-monitoring release cannot publish its source maps. Repeated uploads for the same immutable release are idempotent. This prevents a deployed release from claiming diagnosability it does not provide. The prior release remains available on rollback, and source maps already uploaded for an immutable release remain valid.

## Operational Decomposition

| Workstream | Area of operations | Coordination and sequencing |
|---|---|---|
| Request-failure diagnosis | `internal/observability`, `internal/utils`, request logging tests | Owns the request-scoped failure-record contract. Must complete before release verification because it defines the server event shape. |
| Shared release configuration | `internal/config`, layout metadata, `resources/js/sentry.ts` | Depends on the release-value contract above only for verification naming; preserves the public-config boundary. |
| Browser release assembly | frontend build scripts, container/CI release steps | Depends on the shared release value. Owns Sentry upload credentials and source-map exclusion. |
| Verification | focused Go/JS/build checks and staging Sentry observation | Runs after the preceding workstreams; proves correlation, sanitisation, release equality, map exclusion, and uploaded source resolution. |

The shared failure-record and release-configuration contracts each have one owner during implementation. The browser build/upload work may proceed after the release-value contract is fixed, but deployment changes must be serialised after application changes so the first published release has matching runtime metadata and maps.

## Risks / Trade-offs

- [An originating error contains user data] → Sanitise exception text, use an allowlist for all event/log fields, and test negative assertions with sensitive request values.
- [A handler omits causal failure metadata] → Capture a clearly classified status-only event and use route/request-ID correlation to identify the remaining source; expand the shared server-fault call sites in the same change.
- [Route patterns are absent for a framework-generated response] → Use a single low-cardinality fallback label instead of a URL path.
- [A Sentry upload credential is unavailable] → Fail release assembly before deployment and leave the existing release running.
- [Release metadata diverges between Go and JavaScript] → Derive both from one Docker build argument embedded in the Go binary and verify equality through rendered browser configuration and captured events.
- [Sentry grouping becomes too coarse or too fragmented] → Fingerprint only stable method, route pattern, failure category, and status dimensions; validate with representative failure cases.

## Migration Plan

1. Ensure Railway supplies Git metadata to the Docker build and rotate the leaked source-map upload credential; do not expose the replacement credential to the application or Docker build.
2. Deploy the application and release-assembly changes together to staging.
3. Use the non-production Sentry test route and a controlled server-failure test to confirm release attribution, request/log correlation, and source-map resolution in the separate projects.
4. Promote the immutable release after the checks pass.
5. To roll back, redeploy the previous immutable release value and artefact; its previously uploaded maps remain associated with that release.

## Open Questions

None.
