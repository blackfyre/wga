# Spec Delta

## MODIFIED Requirements

### Requirement: Browser error monitoring

The browser bundle SHALL include the official Sentry browser SDK and SHALL initialise it as the application starts when the shared layout supplies a non-empty DSN. The browser SDK SHALL use the configured WGA environment, SHALL deliver its envelopes through the first-party browser event relay rather than directly to Sentry, and SHALL retain the application's existing browser initialisation behaviour.

#### Scenario: Browser receives a DSN

- **WHEN** a full page is rendered with a configured Sentry DSN
- **THEN** the browser initialises Sentry before the main application bootstrap completes and sends its envelopes to `/diagnostics/browser` on the page's own origin

#### Scenario: Browser receives no DSN

- **WHEN** a full page is rendered with Sentry monitoring disabled
- **THEN** the browser skips Sentry initialisation and the main application bootstrap still completes

## ADDED Requirements

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
