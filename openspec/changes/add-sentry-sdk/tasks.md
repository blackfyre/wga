## 1. Configuration and dependencies

- [x] 1.1 Add `WGA_SENTRY_DSN` as optional, redacted Sentry configuration in `internal/config`, pass it through the server configuration contract, and add focused configuration tests for configured and empty values.
- [x] 1.2 Document `WGA_SENTRY_DSN` in `.env.example` without making it required for local development.
- [x] 1.3 Add the latest compatible `github.com/getsentry/sentry-go` and `@sentry/browser` dependencies, updating the Go and Bun lockfiles.
- [x] 1.4 Reject secret-bearing Sentry DSNs before runtime browser configuration is generated.

## 2. Server monitoring

- [x] 2.1 Create the application-owned observability integration that initialises Sentry with the DSN and WGA environment, logs disabled or failed initialisation, and flushes with a bounded timeout.
- [x] 2.2 Register the integration from the serving composition root and add PocketBase router adaptation that reports unexpected returned errors and panics while preserving existing request handling and excluding expected client errors.
- [x] 2.3 Add unit tests for disabled, failed, and enabled server monitoring paths, including preserved error propagation and redacted logging.

## 3. Browser monitoring

- [x] 3.1 Make only the public Sentry DSN and WGA environment available to the shared Templ layout at runtime, then run `templ generate`.
- [x] 3.2 Initialise `@sentry/browser` from the page configuration before the main browser bootstrap, and skip initialisation when the DSN is empty without interrupting the application.
- [x] 3.3 Add focused browser-facing tests or build assertions covering configured and disabled initialisation paths.
- [x] 3.4 Initialise Sentry before loading the main browser bundle and strip query strings and fragments from browser event URLs.

## 4. Verification

- [x] 4.1 Run `go mod tidy`, `go vet ./...`, and focused Go tests for configuration and observability.
- [x] 4.2 Run `bun run build` and the relevant formatting or browser test checks.
- [ ] 4.3 Manually verify a deployment with no DSN starts normally and logs that monitoring is disabled, then verify configured server and browser test errors arrive in Sentry with the correct environment.
