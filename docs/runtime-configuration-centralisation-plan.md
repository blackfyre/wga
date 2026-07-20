# Runtime Configuration Centralisation Plan

## Introduction

Issue #176 requires one typed runtime-configuration boundary so application code no longer reads environment variables directly. The boundary must validate only settings needed by the active server, sitemap, or migration capability while retaining the existing variable names and local defaults.

## Phase 1: Configuration boundary

### Typed runtime settings

- [✓] Add an immutable configuration package that loads `.env` and process settings once, parses public and storage URLs, SMTP ports, environment names, and the postcard schedule, and returns secret-safe validation errors.
- [✓] Define explicit local/test and protected captcha behaviour. Verification may be skipped only in local/test environments when no secret is configured.

### Capability validation

- [✓] Validate server and sitemap configuration before their commands can execute; migration settings must be validated only by the migration callbacks that use them. Verification: invalid active settings prevent the relevant command without requiring unrelated settings.

## Phase 2: Composition and consumers

### Configuration injection

- [✓] Inject only the relevant configuration subsets into handlers, cron jobs, URL helpers, sitemap generation, and migrations. Verification: no non-configuration Go package directly accesses process environment settings.
- [✓] Preserve migration registration order and schema while resolving settings at callback execution time rather than package initialisation.

### Environment documentation

- [✓] Update `.env.example`, `mise.toml`, and authoritative environment documentation with explicit local defaults and the postcard schedule. Verification: documented keys match the configuration package.

## Phase 3: Verification

### Automated coverage

- [✓] Add focused parsing, validation, migration, and fresh/existing data-directory regression tests. Verification: tests prove capability-specific validation and migration-time configuration resolution.
- [✓] Run formatting, focused tests, `go vet ./...`, and `go test ./... -cover`. Verification: all selected commands pass.

### Review

- [✓] Review the final diff and GitNexus change impact for unintended affected flows.
