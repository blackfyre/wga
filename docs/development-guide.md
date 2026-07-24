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

Use S3-compatible object storage for durable uploaded files. Do not make a local filesystem path the production system of record. Classify new file types as public or restricted; treat an unclassified type as restricted, and require authorisation plus time-bounded access for restricted files.

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
