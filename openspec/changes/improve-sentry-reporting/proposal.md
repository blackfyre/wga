## Why

Current server Sentry events can reduce a written HTTP 500 to a generic status-only exception, leaving operators unable to identify the route, correlate the event with application logs, or determine whether it is a recurring defect. Browser events also lack a release identifier, making deployment-related failures such as missing hashed assets difficult to attribute.

## What Changes

- Enrich unexpected server-failure events with an allowlisted request diagnosis context and correlate them with structured request-failure logs.
- Preserve an available causal server error when reporting a response that is rendered as HTTP 500, without changing the client response.
- Attribute server and browser Sentry events to the deployed release and make browser source maps available to Sentry for that release.
- Retain the existing privacy boundary: monitoring data must not include request bodies, credentials, cookies, query strings, fragments, client IP addresses, or raw user input.
- Define deterministic verification for correlation, grouping, privacy redaction, release attribution, and source-map upload.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `sentry-observability`: enrich server failure reporting and add release/source-map attribution while preserving privacy and response behaviour.
- `separate-sentry-project-configuration`: extend the browser-visible monitoring configuration with release metadata without exposing server credentials.

## Impact

- Affected code: `internal/observability`, request logging, server-fault adapters, `internal/config`, shared layout metadata, `resources/js/sentry.ts`, and the frontend build/release pipeline.
- Affected systems: both WGA Sentry projects, Railway log correlation, and deployment credentials for source-map upload.
- No client-facing route, response, or authentication behaviour should change.
