# Proposal

## Why

Default content blockers stop the browser Sentry SDK from reaching Sentry. Brave shields and the uBlock Origin bundled with Helium both apply EasyPrivacy's `||sentry.io^$3p`, so browser errors from those visitors never arrive, and every page logs `net::ERR_BLOCKED_BY_CLIENT`.

## What Changes

- Add a first-party browser event relay at `POST /diagnostics/browser`. It is registered only when `WGA_SENTRY_BROWSER_DSN` is configured.
- The relay accepts only envelopes addressed to the configured browser project. It forwards them to that project's fixed Sentry ingest URL, and never to a destination derived from the request.
- The relay admits requests only from the canonical public origin and a resolved trusted client identity. It applies a per-client rate limit and a global concurrency limit, and bounds the request size.
- The relay forwards no client address, cookies, or inbound headers. It does not log envelope contents, and it does not report its own upstream failures to server Sentry.
- The browser SDK sends through the relay whenever it initialises with a DSN.
- Out of scope:
  - changing what the browser SDK collects, including session tracking;
  - relaying Rybbit analytics;
  - Cloudflare rule changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `sentry-observability`: browser monitoring delivers through the first-party relay, and a new requirement defines the relay's admission, validation, forwarding, and privacy behaviour.
- `separate-sentry-project-configuration`: the relay never forwards to the server Sentry project.

## Impact

- New `internal/observability/browser_tunnel.go`, the forwarding workflow.
- New `internal/handlers/diagnostics` package, the HTTP adapter and its limiter. It is registered from `internal/handlers/main.go`, and `cmd/wga/main.go` constructs the tunnel.
- `resources/js/sentry.ts` sets the SDK `tunnel` option.
- Go and JS tests.
- `docs/development-guide.md` gains a note on the relay.
- No new configuration settings or dependencies.
