# Spec Delta

## MODIFIED Requirements

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
