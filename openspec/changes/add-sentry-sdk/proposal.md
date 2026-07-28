## Why

The application has no central error-monitoring integration for production failures in either the Go server or browser code. Adding Sentry to both runtimes will make those failures observable while keeping local and unconfigured deployments operational.

## What Changes

- Add the current official Sentry SDKs to the Go server and bundled browser application.
- Configure server-side Sentry from an environment-supplied DSN, including the configured deployment environment.
- Make the browser DSN available to the built frontend without exposing non-public server configuration.
- Report a missing DSN through application logging and leave the relevant runtime running without Sentry.

## Capabilities

### New Capabilities

- `sentry-observability`: Optional Sentry error monitoring for server and browser runtimes.

### Modified Capabilities

None.

## Impact

- `internal/config` and `.env.example` gain Sentry DSN configuration.
- `cmd/wga/main.go`, request handling, and application logging gain server-side monitoring initialisation and reporting.
- Frontend build sources and browser entry points gain Sentry initialisation.
- Go module and JavaScript package manifests and lockfiles gain the official Sentry SDK dependencies.
