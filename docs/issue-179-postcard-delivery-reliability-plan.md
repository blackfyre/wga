# Issue 179 Postcard Delivery Reliability Plan

## Introduction

Issue #179 replaces the current all-or-nothing postcard cron callback with a durable recipient-level delivery workflow. This prevents a failed send from marking a postcard sent, prevents duplicate active work, and makes uncertain SMTP outcomes safe for operator review rather than automatic retry.

## Phase 1: Assess the existing flow

### Current behaviour and constraints

- [✓] Trace postcard submission, cron delivery, pickup rendering, migrations, configuration, and commands. Verification: the current cron unconditionally writes `status=sent`, recipient progress is not stored, and no pickup transition exists.
- [✓] Confirm compatibility constraints. Verification: `/postcard`, `/postcard/send`, the `postcards` collection, existing fields, and `queued`, `sent`, and `received` status values remain unchanged.

## Phase 2: Durable lifecycle foundation

### Schema and feature ownership

- [✓] Add an additive migration for correlation and received timestamps, recipient deliveries, and delivery attempts. Verification: migration tests confirm the new schema while existing postcard fields and status values remain unchanged.
- [✓] Add the `internal/postcards` workflow with explicit parent, delivery, and attempt transitions. Verification: focused tests reject invalid resolution transitions and only every-successful-recipient completion can set the parent sent.
- [✓] Move queue creation behind the workflow. Verification: a new postcard and its normalised, de-duplicated recipient delivery attempts persist together in focused tests.

## Phase 3: Safe scheduled delivery

### Claims, outcomes, and recovery

- [✓] Add atomic expiring claims and claim-token-guarded completion. Verification: focused tests confirm transport start requires the active claim token and stale tokens are rejected.
- [✓] Record SMTP outcomes before terminal parent transitions and retry only classified transient failures with bounded backoff. Verification: focused tests confirm failed attempts leave `status=queued` and `sent_at` empty, while a retry receives a future eligible time.
- [✓] Replace the cron callback with the feature worker. Verification: the cron adapter invokes the bounded feature worker, and legacy queued records expand only into dead-lettered review attempts without sending mail.

## Phase 4: HTTP and operator adapters

### Existing routes and minimal commands

- [✓] Mark a sent postcard received after its first successful pickup-page rendering. Verification: `MarkReceived` transitions sent records once and is idempotent in focused tests.
- [✓] Add inspection, resolution, and replay commands. Verification: commands use safe attempt identifiers, require an explicit replay confirmation, and route state changes through the workflow.

## Phase 5: Verification and rollout

### Automated and operational checks

- [✓] Add migration, lifecycle, claim, retry, recipient, and route tests. Verification: focused migration and workflow tests cover the durable state changes and no longer encode the previous false-sent behaviour.
- [✓] Run focused checks, `go vet ./...`, and `go test ./... -cover`. Verification: focused packages, `go vet ./...`, and the full 39-package Go suite pass.
- [✓] Exercise the postcard journey against Mailpit with Playwright. Verification: `bunx playwright test playwright-tests/postcard.spec.ts` passed against the running application, Mailpit, and Garage services.
- [✓] Commit, push, and open the requested external review artefact. Verification: the Issue #179 branch is pushed and pull request #186 targets `main`.
