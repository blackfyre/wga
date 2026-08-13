## 1. Separate Sentry configuration

- [x] 1.1 Extend `internal/config` with optional `WGA_SENTRY_BROWSER_DSN`, independently validate it, retain redaction, and add focused tests for separate, absent, and secret-bearing DSNs.
- [x] 1.2 Document the separate server and browser DSN settings in `.env.example` without adding either value to source control.

## 2. Independent runtime configuration

- [x] 2.1 Update observability setup so server SDK initialisation uses only `WGA_SENTRY_DSN`, while browser page configuration always uses `WGA_SENTRY_BROWSER_DSN` and the environment.
- [x] 2.2 Update the shared page configuration and focused tests to prove that only the browser DSN is rendered and browser initialisation remains independent of server monitoring.

## 3. Verification

- [x] 3.1 Run `templ generate`, focused Go configuration and observability tests, `go vet ./...`, and `bun run build`.
- [x] 3.2 In non-production deployment configuration, set the server and supplied browser DSNs, visit `/sentry-test`, and verify each event reaches its respective Sentry project with the correct environment.
