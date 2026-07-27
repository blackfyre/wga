# Issue 178 Owned Integration Boundaries Plan

## Introduction

Issue #178 removes GitHub fetching from the contributors request path and moves reCAPTCHA verification behind an application-owned boundary. The contributor refresh must retain a durable last-known-good snapshot and execution metadata, while captcha remains synchronous so rejected or unavailable verification cannot create a postcard.

## Phase 1: Assess the existing flow

### Request, provider, and persistence boundaries

- [✓] Trace contributors, captcha, cron, configuration, migration, and delivery patterns. Verification: the contributors route currently invokes GitHub on a cache miss, and postcard persistence calls the default HTTP client directly.
- [✓] Confirm compatibility constraints. Verification: `/contributors`, its rendered page, `X-WGA-Contributors-Source`, and postcard rejection/provider-failure response classes remain unchanged.

## Phase 2: Establish owned boundaries

### Contributor snapshots and scheduled refresh

- [✓] Add a feature-owned GitHub adapter, durable snapshot store, fallback reader, and bounded refresh workflow. Verification: the route reads only a snapshot or fallback and refresh records an outcome before and after every external attempt without replacing a prior snapshot on failure.
- [✓] Add an additive migration for contributor snapshots and refresh executions. Verification: migration tests confirm the collections, indexes, rollback, and reapplication behaviour.
- [✓] Register the configured contributor adapter and named refresh job at application startup. Verification: the cron adapter invokes the workflow with a fresh run identifier and the route cannot receive a GitHub client.

### Synchronous anti-abuse verification

- [✓] Move reCAPTCHA transport and response mapping into an application-owned verifier. Verification: the configured client has a strict timeout, cancellation propagates, and provider payloads and statuses do not cross the verifier boundary.
- [✓] Inject the verifier into postcard handling. Verification: accepted captcha queues a postcard; rejection, timeout, and provider failure return before persistence with the existing client-response classes.

## Phase 3: Verification and delivery

### Focused regression coverage

- [✓] Add provider, fallback, stale snapshot, refresh metadata, route, captcha, and persistence-prevention tests. Verification: controlled HTTP servers cover success, rejection, timeout, malformed responses, and provider outages.
- [✓] Run focused tests, `go vet ./...`, and `go test ./... -cover`. Verification: all selected checks pass.
- [✓] Commit, push, and open a pull request for issue #178. Verification: the branch is pushed and pull request #192 targets `main` with a Conventional Commit title.
