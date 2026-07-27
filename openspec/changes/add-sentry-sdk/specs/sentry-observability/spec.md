## ADDED Requirements

### Requirement: Optional Sentry configuration

The application SHALL load `WGA_SENTRY_DSN` through `internal/config` as optional server configuration and SHALL document the setting in `.env.example`. An empty value SHALL be valid and SHALL not cause configuration validation or application startup to fail.

#### Scenario: DSN is configured

- **WHEN** `WGA_SENTRY_DSN` contains a Sentry DSN
- **THEN** the serving runtime receives that value as its Sentry configuration

#### Scenario: DSN is omitted

- **WHEN** `WGA_SENTRY_DSN` is unset or empty
- **THEN** the application starts normally and emits one structured log entry that Sentry monitoring is disabled

#### Scenario: DSN is malformed

- **WHEN** `WGA_SENTRY_DSN` is supplied in an invalid format
- **THEN** server configuration fails with an operator-safe validation error that identifies `WGA_SENTRY_DSN`

#### Scenario: DSN contains a secret key

- **WHEN** `WGA_SENTRY_DSN` contains a password or secret key
- **THEN** server configuration rejects the setting before it can be exposed to browser code

### Requirement: Server error monitoring

The serving runtime SHALL initialise the official Sentry Go SDK when a valid DSN is configured, SHALL label events with the configured WGA environment, and SHALL use bounded flushing during shutdown. It SHALL capture unhandled server failures without changing the response or error-propagation behaviour observed by clients.

#### Scenario: Unhandled server error

- **WHEN** a request handler returns an unexpected server error while Sentry is configured
- **THEN** the error is reported to Sentry and the existing client response behaviour is preserved

#### Scenario: Server panic

- **WHEN** request processing panics while Sentry is configured
- **THEN** the panic is reported to Sentry and the application's existing panic handling remains in effect

#### Scenario: SDK initialisation fails

- **WHEN** Sentry cannot initialise from the configured DSN
- **THEN** the failure is logged and the application continues without Sentry monitoring

### Requirement: Intentional Sentry verification

The application SHALL provide a non-production route that sends the message `It works!` from the server and browser. The server event SHALL be flushed before the route responds. The route SHALL not be registered in production and SHALL report disabled monitoring instead of claiming delivery.

#### Scenario: Non-production test events

- **WHEN** an operator visits `/sentry-test` with Sentry monitoring configured outside production
- **THEN** the application sends `It works!` to Sentry from both the server and browser

#### Scenario: Production test route

- **WHEN** an operator requests `/sentry-test` in production
- **THEN** the application does not register the route

#### Scenario: Disabled non-production monitoring

- **WHEN** an operator requests `/sentry-test` without Sentry monitoring configured
- **THEN** the application reports that Sentry monitoring is disabled

### Requirement: Browser error monitoring

The browser bundle SHALL include the official Sentry browser SDK and SHALL initialise it as the application starts when the shared layout supplies a non-empty DSN. The browser SDK SHALL use the configured WGA environment and SHALL retain the application's existing browser initialisation behaviour.

#### Scenario: Browser receives a DSN

- **WHEN** a full page is rendered with a configured Sentry DSN
- **THEN** the browser initialises Sentry before the main application bootstrap completes

#### Scenario: Browser receives no DSN

- **WHEN** a full page is rendered with Sentry monitoring disabled
- **THEN** the browser skips Sentry initialisation and the main application bootstrap still completes

### Requirement: Browser event privacy

The browser SDK SHALL remove query strings and fragments from error-event and breadcrumb URLs, and SHALL remove console breadcrumb arguments, before sending events to Sentry.

#### Scenario: Page URL contains a sensitive query parameter

- **WHEN** a browser error occurs on a URL containing query parameters or a fragment
- **THEN** the event sent to Sentry contains the URL path without the query string or fragment

#### Scenario: Console breadcrumb contains form data
- **WHEN** a console breadcrumb contains logged form data
- **THEN** the event sent to Sentry excludes the breadcrumb arguments

### Requirement: Public configuration boundary

The application SHALL expose to browser code only the public Sentry DSN and deployment environment required for browser monitoring. It SHALL NOT expose unrelated server configuration or secrets through the layout, static assets, or monitoring events.

#### Scenario: Rendered page configuration

- **WHEN** a page is rendered with Sentry configured
- **THEN** its browser-visible monitoring configuration contains the DSN and environment only
