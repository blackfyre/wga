# Tasks

## 1. Forwarding workflow

- [x] 1.1 Add `internal/observability/browser_tunnel.go`: construction from the browser and server DSNs (nil when the browser DSN is empty), first-line envelope DSN validation, and forwarding to the precomputed ingest URL with a dedicated no-redirect client. Cover the DSN matrix (scheme, host, port, path, key, project, server DSN, missing or malformed header), header allowlists in both directions, redirects, and timeout and transport-error mapping with a fake `RoundTripper` asserting zero upstream calls on rejection. Verify with `go test ./internal/observability -run 'BrowserTunnel'`.

## 2. HTTP adapter

- [x] 2.1 Add `internal/handlers/diagnostics` with `POST /diagnostics/browser`. It covers the body size limit, identity-only content encoding, canonical-origin check, trusted identity, HMAC-keyed per-client limiter, capacity gate with `Retry-After`, result mapping, `Cache-Control: no-store`, `MarkExpectedResponse` on 5xx, and outcome-only logging. Register it from `internal/handlers/main.go`, and build the tunnel in `cmd/wga/main.go`. Cover each rejection path, successful relay, rate and capacity, and the unregistered route without a browser DSN. Verify with `go test ./internal/handlers/diagnostics ./internal/handlers` and `go vet ./...`.

## 3. Browser client

- [x] 3.1 Set `tunnel: "/diagnostics/browser"` in `initialiseSentry` and extend `resources/js/sentry.test.ts` to assert it. Verify with `bun test resources/js/sentry.test.ts` and `bun run build`.

## 4. Verification and documentation

- [x] 4.1 Add a Playwright check that, with a browser DSN configured, a browser event is posted to `/diagnostics/browser` on the page's own origin and no request goes to `sentry.io`. Verify with `mise run test:playwright`.
- [x] 4.2 Document the relay in `docs/development-guide.md` per `docs/documentation-maintenance.md`. Then run `go mod tidy`, `go vet ./...`, `go test ./... -cover`, and golangci-lint (`--new-from-rev=HEAD`) with passing results.
