## Why

The server and browser report to separate Sentry projects, but the current integration shares one DSN. The browser needs its own public DSN so events are routed to its project without changing server monitoring.

## What Changes

- Add optional configuration for a browser-specific Sentry DSN, separate from the existing server DSN.
- Supply only the browser DSN and deployment environment to rendered browser pages.
- Preserve disabled monitoring when either runtime's DSN is absent and keep server events on the server DSN.
- Document the two DSN settings and verify server and browser test events reach their respective Sentry projects.

## Capabilities

### New Capabilities

- `separate-sentry-project-configuration`: Independent optional Sentry DSNs for server and browser monitoring.

### Modified Capabilities

None.

## Impact

- `internal/config` and `.env.example` gain browser-specific Sentry configuration.
- `internal/observability` and shared page rendering use the browser DSN rather than the server DSN.
- Configuration and browser-facing tests cover independent runtime settings.
