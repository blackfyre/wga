# Sentry Observability

## Purpose

Define optional server and browser Sentry monitoring, intentional verification, and browser-event privacy controls.

## Requirements

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

The browser bundle SHALL include the official Sentry browser SDK and SHALL initialise it as the application starts when the shared layout supplies a non-empty DSN. The browser SDK SHALL use the configured WGA environment, SHALL deliver its envelopes through the first-party browser event relay rather than directly to Sentry, and SHALL retain the application's existing browser initialisation behaviour.

#### Scenario: Browser receives a DSN

- **WHEN** a full page is rendered with a configured Sentry DSN
- **THEN** the browser initialises Sentry before the main application bootstrap completes and sends its envelopes to `/diagnostics/browser` on the page's own origin

#### Scenario: Browser receives no DSN

- **WHEN** a full page is rendered with Sentry monitoring disabled
- **THEN** the browser skips Sentry initialisation and the main application bootstrap still completes

### Requirement: Browser event privacy

The browser SDK SHALL remove query strings and fragments from error-event, stack-frame, and breadcrumb URLs, and SHALL remove console breadcrumb arguments, before sending events to Sentry.

#### Scenario: Page URL contains a sensitive query parameter

- **WHEN** a browser error occurs on a URL containing query parameters or a fragment
- **THEN** the event sent to Sentry contains the URL path without the query string or fragment

#### Scenario: Stack frame contains a sensitive query parameter

- **WHEN** a browser error stack frame contains a URL with query parameters or a fragment
- **THEN** the stack-frame URL sent to Sentry contains no query string or fragment

#### Scenario: Console breadcrumb contains form data

- **WHEN** a console breadcrumb contains logged form data
- **THEN** the event sent to Sentry excludes the breadcrumb arguments

### Requirement: Public configuration boundary

The application SHALL expose to browser code only the public Sentry DSN and deployment environment required for browser monitoring. It SHALL NOT expose unrelated server configuration or secrets through the layout, static assets, or monitoring events.

#### Scenario: Rendered page configuration

- **WHEN** a page is rendered with Sentry configured
- **THEN** its browser-visible monitoring configuration contains the DSN and environment only

### Requirement: Browser event relay

When `WGA_SENTRY_BROWSER_DSN` is configured, the application SHALL accept browser Sentry envelopes at `POST /diagnostics/browser` and SHALL forward each accepted envelope unchanged to the ingest endpoint of the configured browser Sentry project. The relay's destination SHALL be derived only from configuration. The relay SHALL accept only envelopes whose header DSN identifies the configured browser project. It SHALL admit only requests from the canonical public origin that carry a resolved trusted client identity, SHALL bound request size, per-client request rate, and concurrent forwarding, and SHALL answer rejected requests without contacting Sentry. It SHALL NOT forward the client's address, cookies, or inbound request headers. It SHALL NOT record envelope contents in logs, and SHALL NOT report its own upstream failures to server Sentry.

#### Scenario: Relay is not configured

- **WHEN** `WGA_SENTRY_BROWSER_DSN` is absent
- **THEN** `POST /diagnostics/browser` is not handled by the relay

#### Scenario: Valid envelope is relayed

- **WHEN** a same-origin request with a trusted client identity posts an envelope whose header DSN matches the configured browser DSN
- **THEN** the envelope body is forwarded to the configured browser project's ingest endpoint, and the response carries Sentry's status together with its `X-Sentry-Rate-Limits` and `Retry-After` headers

#### Scenario: Envelope targets another project

- **WHEN** an envelope's header DSN differs from the configured browser DSN in scheme, host, port, path, public key, or project, or its header is missing or malformed
- **THEN** the relay rejects the request and does not contact Sentry

#### Scenario: Oversized or encoded request

- **WHEN** a request body exceeds the relay's size limit or declares a content encoding other than identity
- **THEN** the relay rejects the request and does not contact Sentry

#### Scenario: Cross-origin request

- **WHEN** a request carries an `Origin` or `Referer` that does not match the canonical public origin
- **THEN** the relay rejects the request and does not contact Sentry

#### Scenario: Untrusted client identity

- **WHEN** the relay cannot resolve a trusted client identity for the request
- **THEN** the relay rejects the request and does not contact Sentry

#### Scenario: Client exceeds its budget

- **WHEN** a client exceeds the relay's per-client rate, or the relay's concurrent forwarding capacity is exhausted
- **THEN** the relay answers with a retryable status and `Retry-After` and does not contact Sentry

#### Scenario: Sentry is unreachable

- **WHEN** forwarding fails or times out
- **THEN** the relay answers with a gateway error, logs only the outcome, and does not report the failure to server Sentry

#### Scenario: Forwarded request carries no client data

- **WHEN** an envelope is forwarded
- **THEN** the forwarded request contains the envelope body and content type only, without the client's address, cookies, or inbound headers, and no relay log entry contains the envelope body or header
