## 1. Request-failure diagnosis

- [x] 1.1 Extend the shared server-fault response boundary to retain an optional stable failure category and causal error on the request event, and update its call sites without changing existing 5xx responses; verify focused helper and handler tests preserve status and rendered content.
- [x] 1.2 Enrich server Sentry capture through an isolated scope with the allowlisted request diagnosis, causal-error preference, fail-closed sanitisation for errors and panics, low-cardinality fingerprint, and status-only fallback; verify `go test ./internal/observability ./internal/logging ./internal/utils` proves event fields, correlation, grouping, and absence of sensitive values.
- [x] 1.3 Emit one request-scoped `observability.request.failed` structured log for every captured unexpected server failure; verify integration tests prove its request ID and diagnosis fields match the captured Sentry event.

## 2. Shared release attribution

- [x] 2.1 Embed the Docker `WGA_RELEASE` build argument in `buildinfo.Version` and propagate it to server Sentry configuration and browser layout metadata without exposing server DSNs or other secrets; verify focused configuration, observability, layout, and browser Sentry tests cover enabled and disabled monitoring.
- [x] 2.2 Configure browser Sentry initialisation to identify events with the shared release value and fail-closed URL/console-breadcrumb scrubbing; verify the browser test suite proves the configured release is passed to the SDK and sensitive malformed URLs, console messages, and arguments are excluded.

## 3. Source-map release assembly

- [ ] 3.1 Extend the frontend release build to generate browser source maps in a clean non-public staging directory, retain them only in the final image, and upload them idempotently from the container entrypoint using a sealed Railway runtime token and embedded release; verify a deterministic build fixture checks the upload inputs and confirms no `.map` files exist in `internal/assets/public`.
- [ ] 3.2 Document and provision the protected release identifier and Sentry upload credential in the deployment environment without adding either credential to browser-visible configuration; verify deployment configuration exposes only the intended public release metadata at runtime.

## 4. End-to-end verification

- [ ] 4.1 Run `templ generate`, `bun run build`, the affected Go and browser tests, `go vet ./...`, and `openspec validate improve-sentry-reporting --strict`; verify all checks pass and generated source maps are not publicly embedded.
- [ ] 4.2 In staging, trigger the non-production Sentry test route and a controlled server failure; verify the separate server and browser projects show the same release, the server event correlates with its structured log by request ID, and a browser stack frame resolves to authored source.
