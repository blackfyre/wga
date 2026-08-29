# Development Guide

## Purpose

This guide records durable constraints for changes to WGA. It complements the repository workflow in `AGENTS.md` and the maintenance checklist in `docs/documentation-maintenance.md`.

## Application structure

WGA is one deployable application with feature-level boundaries. Keep related handlers, persistence access, scheduled work, and external adapters close to the capability that owns them. Do not introduce a separate service merely to organise a feature.

`cmd/wga/main.go` is the composition root. It creates the application and registers routes, hooks, cron jobs, and migrations before starting the server. New capabilities should extend their owning package and be registered from the established entry points.

When a change crosses capability boundaries, prefer an explicit input, query, or event contract over reaching into another capability's persistence helpers. Keep dependencies acyclic where practical.

## HTTP, hooks, and business workflows

Handlers and hooks are framework adapters. They should parse and validate transport input, obtain request context, invoke the owning workflow, and map the result to a response or framework action.

Keep non-trivial workflow orchestration, product rules, state transitions, and external side-effect ordering outside request handlers. The same rule should remain reusable from a cron job, hook, or command without depending on an HTTP request object.

Keep application-level failures distinct from malformed requests and framework failures. Do not leak framework, database, or provider errors directly into user-facing responses.

## Configuration and storage

Load deployment configuration through `internal/config`. Feature code must not read environment variables or `.env` files directly. Add parsing, validation, normalisation, and focused tests to that package when adding a setting.

Configuration is resolved at process startup. Required settings must fail validation before the application serves traffic or starts scheduled work. Keep secrets out of errors, debug output, and logs.

Postcard delivery requires `WGA_POSTCARD_TOKEN_KEYS`, a secret JSON object mapping key IDs to unpadded Base64URL-encoded 32-byte keys, and `WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID`, which names the key used for new envelopes. Generate each key independently with `python3 -c 'import base64, secrets; print(base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode())'`, then inject the resulting value through the deployment secret mechanism rather than committing it to `.env.example`. Server startup rejects a missing or invalid keyring; errors, logs, and operator commands must not print keys, bearer tokens, or token envelopes.

Rotate postcard token keys in this order:

1. Add the new key under a new key ID while retaining every key referenced by live envelopes.
2. Set `WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID` to the new ID and restart the server so newly queued postcards use it.
3. Run `dist/wga postcards rewrap-token-key --from <old-id>` against the intended data directory; use `--limit` to bound a smaller batch when required, then repeat until the aggregate count is zero.
4. Verify the command reports only aggregate counts, no live envelope still references the old ID, and postcard delivery remains healthy.
5. Take and verify the required backup after rewrapping.
6. Retain the old key until every backup that can restore an old-key envelope has passed its retention window.
7. Remove the old key, restart, and confirm configuration validation and postcard delivery again.

WGA's current data-directory model resets the database between runs. Treat the baseline schema migrations as the complete contract for a fresh database: update the baseline field definition when a field's current shape changes, rather than adding a migration solely to modify historical state that is never retained.

Use S3-compatible object storage for durable uploaded files. Do not make a local filesystem path the production system of record. Classify new file types as public or restricted; treat an unclassified type as restricted, and require authorisation plus time-bounded access for restricted files.

Anonymous-write admission (visitor itineraries, and future participation surfaces) resolves a trusted client identity through `WGA_CLIENT_IP_SOURCE`. `direct` parses and canonicalises the socket peer (`RemoteAddr`) and ignores forwarding headers; `railway` is the production Railway-edge contract and requires exactly one syntactically valid `X-Railway-Edge` marker plus exactly one parseable `X-Real-IP`, ignores `X-Forwarded-For`, and fails closed on anything else. Development and test default to `direct`; production and staging must select a source explicitly. The resolver (`internal/requesttrust`) neither persists nor logs the raw client address — it is hashed by callers before use in admission limits.

## Scheduled and external work

Cron jobs and other non-request executions start a fresh `run_id`. Preserve a stable correlation value when work continues an earlier flow, but do not reuse the originating execution identifier as the current run identifier.

For a new direct third-party integration, record work durably before attempting the external side effect when traceability, recovery, or retry matters. Define duplicate suppression or idempotency before automatic retries can repeat an external action. Make terminal outcomes explicit and retain only the data required to operate and diagnose the integration.

Classify failures at the adapter that understands the dependency. Retry only failures that can succeed unchanged; leave deterministic input, credential, configuration, contract, and code failures for explicit resolution rather than retrying them indefinitely.

## Logging and personal data

Use structured log attributes, not interpolated diagnostic strings. Attach the request-scoped logger for HTTP work and the run-scoped logger for cron or background work. WGA generates request identifiers at ingress and does not trust public request headers as an identifier source.

Use `logging.Redact` for values that may contain personal data, credentials, tokens, or payload content. Prefer internal record identifiers over names, email addresses, addresses, or raw user input. Logs and operational records must retain the minimum data needed for support and must not become a substitute for durable business history.

When a feature stores personal data, define its purpose and retention outcome. Remove or anonymise data that is no longer needed, including copies in uploads, exports, and operational by-products.

## Delivery discipline

Use Conventional Commit types for commits and pull-request titles. Keep documentation aligned with the executable configuration and CI workflow; task plans, review notes, and historical summaries are not a substitute for current guidance.

## Licence notices and SBOM

`internal/licences/manifest.json` is the reviewed record of every third-party component shipped by WGA. It contains source evidence, licence text, NOTICE material, integrity data, distribution target, and dependency relationships for the reviewed component version.

`cmd/generate-licences` discovers Go modules with `go list -deps -json ./cmd/wga`, JavaScript packages from `dist/browser-metafile.json`, browser-CSS imports, and declared third-party code bundled inside package artefacts. It validates discovery against the manifest, then writes the embedded notice page at `internal/assets/views/open-source-licences.html` and the release artefact at `dist/wga.cdx.json`.

Run `go run ./cmd/generate-licences` after `bun run build` and `go tool templ generate`. When a dependency changes, run `go run ./cmd/generate-licences --bootstrap`, then review every changed SPDX identifier, full licence text, and required NOTICE or attribution material before committing the manifest and notice page. Do not edit generated HTML or SBOM output directly. `mise run app:build` and GoReleaser run the generator automatically; release archives include `wga.cdx.json` alongside the binary.

## FOSSA analysis

`.fossa.yml` scopes FOSSA analysis to the root Go and frontend dependency manifests and excludes `.opencode` tooling metadata. Use `fossa list-targets` only to identify candidate targets: it does not apply configured filters. Use `fossa analyze --output --json` to verify the configured analysis without uploading a revision.

`docs/fossa-licensing-evidence.md` records the current compiled-source findings for `modernc.org/libc@v1.75.6` and `modernc.org/sqlite@v1.57.0`. The findings remain unresolved pending legal review. Do not create a FOSSA exception, policy approval, licence-data correction, or credentialed CI policy gate from that evidence. Repeat the FOSSA match and build-selection review whenever either module version changes.
