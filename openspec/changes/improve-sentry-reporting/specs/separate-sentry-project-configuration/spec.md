## MODIFIED Requirements

### Requirement: Browser configuration excludes the server DSN
The application SHALL expose only the public browser Sentry DSN, deployment environment, and deployed release to rendered browser pages. It SHALL NOT expose the server Sentry DSN through page markup, static assets, or browser monitoring events.

#### Scenario: Separate DSNs are configured

- **WHEN** a full page is rendered with distinct server and browser Sentry DSNs
- **THEN** the browser-visible monitoring configuration contains the browser DSN, environment, and release but not the server DSN

#### Scenario: Browser monitoring is disabled

- **WHEN** a full page is rendered without `WGA_SENTRY_BROWSER_DSN`
- **THEN** the browser-visible monitoring configuration has no DSN and the main browser bootstrap still completes
