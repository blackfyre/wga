## Context

WGA currently embeds a Bun-built browser bundle in its Go binary and starts PocketBase from `cmd/wga/main.go`. Runtime settings are loaded solely through `internal/config`; browser source cannot read those settings after the binary has been built. Errors are currently logged by PocketBase and feature handlers but are not sent to an error-monitoring service.

The Sentry DSN is public client configuration, but it must remain a runtime deployment setting so the same built binary can be configured without rebuilding. An empty DSN is a supported deployment state and must not make `serve` fail.

## Goals / Non-Goals

**Goals:**

- Initialise the current official Sentry Go and browser SDKs from one optional DSN environment setting.
- Capture unhandled server request failures and browser errors without changing normal response behaviour.
- Associate events with the configured WGA environment and flush server events during shutdown.
- Expose only the DSN required by the browser at runtime, and log once when monitoring is disabled.

**Non-Goals:**

- Adding performance tracing, session replay, profiling, user identity, or custom business-event instrumentation.
- Treating a missing Sentry configuration as a startup failure.
- Sending server secrets or request bodies to Sentry.

## Decisions

### Own integration behind a WGA observability package

Create a small application-owned observability package that accepts an explicit configuration contract from `internal/config`, initialises the Go SDK, registers request error/panic capture, and exposes the public browser configuration needed by the shared page layout. `cmd/wga/main.go` remains the composition root.

This keeps framework adaptation and external side-effect ordering out of handlers, centralises enablement, and avoids allowing feature code to access environment variables. Direct SDK calls scattered through handlers were rejected because they would duplicate configuration and make error coverage inconsistent.

### Use a single optional `WGA_SENTRY_DSN` runtime setting

Add the DSN to `internal/config` as optional server configuration, preserving its redacted representation in logs. An omitted DSN disables monitoring, while a supplied malformed or secret-bearing DSN is rejected during configuration validation. The same valid public DSN is supplied to the Go and browser SDKs. The configured `WGA_ENV` is passed as the Sentry environment.

The DSN is intentionally not embedded by the Bun build: build-time substitution would require a separate build for each deployment and would not honour the runtime `.env` loaded by the Go process.

### Publish browser configuration through the shared layout

The observability package supplies the public DSN to the common Templ layout at request-render time. The layout emits it in a dedicated metadata/configuration element, and the browser entry point reads that value before initialising `@sentry/browser`.

This makes the configuration available as the application starts in the browser without an extra request or per-page handler changes. A runtime configuration endpoint was rejected because it delays initialisation and adds a public route solely for static process configuration.

The browser entry bundle initialises Sentry before dynamically loading the remainder of the application. Its `beforeSend` hook strips query strings and fragments from event and breadcrumb URLs so bearer-style query parameters are not sent to Sentry.

### Capture only unexpected server failures and preserve PocketBase semantics

Register a request-level framework adapter around the existing router that captures returned unexpected errors and recovered panics, then preserves the existing error propagation and response handling. Client-side validation and other expected 4xx responses are not reported as server errors. The SDK client is flushed with a bounded timeout when the serving process exits.

This uses the application router rather than a foreign HTTP wrapper so existing PocketBase hooks, request IDs, routing, and error pages remain intact.

### Disabled monitoring logs once and continues

When `WGA_SENTRY_DSN` is empty, server startup emits one structured warning that monitoring is disabled, then registers no Sentry client or capture middleware. The browser detects the absent page configuration and does not initialise Sentry. SDK initialisation errors are similarly logged and leave the application running.

This is preferable to failing startup because telemetry is non-essential and the requested disabled state must keep the service available.

## Risks / Trade-offs

- [A browser DSN is visible in page source] → A Sentry DSN is public ingestion configuration; expose only the DSN, never credentials or other environment values.
- [Browser URLs can contain bearer-style query parameters] → Strip query strings and fragments from error-event and breadcrumb URLs before sending them to Sentry.
- [Automatic browser capture can increase bundle size and event volume] → Use only the browser SDK's default error capture; do not enable tracing, replay, or profiling.
- [Middleware can alter error handling if it consumes errors] → Capture then return/rethrow the original failure, with focused tests for the existing error response path.
- [Buffered events can be lost during shutdown] → Flush with a bounded timeout; shutdown must never block indefinitely.
- [Invalid DSN can prevent SDK initialisation] → Log the SDK error and continue without monitoring.

## Migration Plan

1. Deploy the code with `WGA_SENTRY_DSN` omitted; verify the one disabled-monitoring log entry and unchanged application availability.
2. Set `WGA_SENTRY_DSN` in a non-production environment, deploy, and trigger controlled server and browser errors to verify events carry the environment tag.
3. Promote the DSN to production after event filtering and ownership are confirmed.
4. To roll back, remove the integration release or unset `WGA_SENTRY_DSN`; no data migration or schema rollback is required.

## Verification Trigger

Visit `/sentry-test` in a non-production environment with `WGA_SENTRY_DSN` configured to send the intentional `It works!` message from both the server and browser. The route is not registered in production and reports disabled monitoring instead of claiming delivery.

## Open Questions

None.
