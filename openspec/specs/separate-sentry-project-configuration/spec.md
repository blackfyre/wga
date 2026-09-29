# Separate Sentry Project Configuration

## Purpose

Define independent server and browser Sentry configuration without exposing server credentials to browser code.

## Requirements

### Requirement: Independent runtime Sentry DSNs

The application SHALL load `WGA_SENTRY_DSN` as the optional server Sentry DSN and `WGA_SENTRY_BROWSER_DSN` as the optional browser Sentry DSN through `internal/config`. It SHALL validate each DSN independently and SHALL allow either setting to be absent without preventing application startup.

#### Scenario: Both runtime DSNs are configured

- **WHEN** `WGA_SENTRY_DSN` and `WGA_SENTRY_BROWSER_DSN` contain valid, different DSNs
- **THEN** server monitoring uses the server DSN and browser monitoring receives the browser DSN

#### Scenario: Only the server DSN is configured

- **WHEN** `WGA_SENTRY_DSN` contains a valid DSN and `WGA_SENTRY_BROWSER_DSN` is absent
- **THEN** server monitoring is enabled and browser monitoring is disabled

#### Scenario: Only the browser DSN is configured

- **WHEN** `WGA_SENTRY_BROWSER_DSN` contains a valid DSN and `WGA_SENTRY_DSN` is absent
- **THEN** browser monitoring is enabled and server monitoring is disabled without preventing application startup

#### Scenario: A runtime DSN is secret-bearing

- **WHEN** either Sentry DSN contains a password or secret key
- **THEN** server configuration rejects that setting before it can initialise an SDK or be exposed to browser code

### Requirement: Browser configuration excludes the server DSN

The application SHALL expose only the public browser Sentry DSN and deployment environment to rendered browser pages. It SHALL NOT expose the server Sentry DSN through page markup, static assets, or browser monitoring events, and the browser event relay SHALL NOT forward any envelope to the server Sentry project.

#### Scenario: Separate DSNs are configured

- **WHEN** a full page is rendered with distinct server and browser Sentry DSNs
- **THEN** the browser-visible monitoring configuration contains the browser DSN and environment but not the server DSN

#### Scenario: Browser monitoring is disabled

- **WHEN** a full page is rendered without `WGA_SENTRY_BROWSER_DSN`
- **THEN** the browser-visible monitoring configuration has no DSN and the main browser bootstrap still completes

#### Scenario: Relayed envelope names the server project

- **WHEN** a browser envelope posted to the relay carries the server Sentry DSN in its header
- **THEN** the relay rejects the envelope and does not contact Sentry
