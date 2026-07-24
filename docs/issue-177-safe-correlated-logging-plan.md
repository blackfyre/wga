# Issue 177 Safe Correlated Logging Plan

## Introduction

Issue #177 removes personally identifiable and secret request data from application logs while making request and postcard-delivery logs correlatable. PocketBase persists structured `slog` attributes as JSON log data, so the implementation will use stable fields and local logger instances rather than a separate logging backend.

## Phase 1: Correlation and redaction foundation

### Request and run scopes

- [✓] Inspect the existing handler, cron, hook, and contributor logging paths. Verification: each unsafe or uncorrelated logging call is identified before editing.
- [✓] Add request-ID middleware at the global PocketBase router ingress, including request context propagation and an `X-Request-ID` response header. Verification: one request retains the same non-empty `request_id` in its context and structured logs.
- [✓] Add request/run logger helpers and an explicit redaction utility. Verification: emitted fields use stable names and redacted values never contain their inputs.

## Phase 2: Safe application logs

### Request handlers and integrations

- [✓] Replace postcard handler logs with event names, safe outcomes, and request-scoped loggers. Verification: captured postcard logs do not contain form data, captcha tokens, email addresses, message bodies, IP addresses, or postcard pickup identifiers.
- [✓] Pass correlation context through the contributors repository and return generic contributor failures. Verification: contributor diagnostics are redacted and HTTP responses contain no internal error text.
- [✓] Replace file-hook event serialisation with selected safe fields. Verification: captured file-download logs exclude the request object, paths, names, headers, and body data.

### Scheduled delivery

- [✓] Generate one `run_id` per postcard cron callback and include it in delivery, rendering, record-update, and run-result logs. Verification: all captured delivery records for a run share its `run_id`, include only safe execution IDs, attempt, and outcome fields, and failed deliveries remain queued.

## Phase 3: Verification and delivery

### Automated coverage

- [✓] Add focused identifier-propagation, captured-log redaction, delivery-field, and generic HTTP-error tests. Verification: known sensitive markers fail the tests if they appear in captured logs or contributor responses.
- [ ] Run formatting, focused tests, `go vet ./...`, and `go test ./... -cover`. Verification: all selected commands pass. Blocked: the unrelated `TestConfigurationParsesTypedValues` still expects SMTP port 1025 while the working tree changes its fixture to 1525.

### Review

- [ ] Manually check the changed log field names and contributor error response. Verification: operator-visible output meets the issue acceptance criteria.
- [✓] Commit and push the approved change. Verification: commit contains only issue #177 files and no credentials or generated artefacts.
