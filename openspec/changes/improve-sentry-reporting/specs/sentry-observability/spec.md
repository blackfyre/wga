## ADDED Requirements

### Requirement: Request failure diagnosis
For every unexpected server failure reported to Sentry, the application SHALL attach an allowlisted diagnosis context containing the request ID, HTTP method, normalised route pattern, response status, failure category, and a stable issue fingerprint. It SHALL emit a structured request-failure log with the same request ID and diagnostic fields. It SHALL preserve and report an available causal error when a request produces a server-error response, without changing the response observed by the client.

#### Scenario: Handler returns an unexpected server error
- **WHEN** a request handler returns an unexpected server error
- **THEN** Sentry receives the causal error with the request diagnosis context and the application emits a matching structured request-failure log

#### Scenario: Handler renders a server-error response with a causal error
- **WHEN** a request renders an HTTP 500 response after recording a causal error
- **THEN** Sentry reports that causal error rather than a status-only exception and the client receives the existing response

#### Scenario: Server-error response has no causal error
- **WHEN** a request completes with an HTTP 5xx response and no causal error is available
- **THEN** Sentry receives a status-only event marked as lacking a causal error, with the request diagnosis context and a matching structured request-failure log

#### Scenario: Client cancels a request
- **WHEN** request processing ends because the client cancels the request or its context deadline expires
- **THEN** the application does not report it as an unexpected server failure

#### Scenario: Sensitive request data is present
- **WHEN** a reported server failure includes a request with query parameters, fragments, cookies, credentials, request bodies, client IP addresses, or user-supplied content
- **THEN** none of those values are sent in the Sentry event or structured request-failure log

#### Scenario: Error or panic value contains sensitive data
- **WHEN** a returned error, recorded causal error, wrapped error, or panic value contains sensitive data
- **THEN** the Sentry event retains only safe diagnostic classification and stack information without exporting the sensitive value

### Requirement: Release attribution and browser source maps
The application SHALL identify server and browser Sentry events with the same deployed release identifier. Each deployed browser release with Sentry monitoring enabled SHALL make matching source maps available to the browser Sentry project, so that a minified browser stack frame can be resolved to the authored source. Source maps SHALL not be publicly served with the deployed application.

#### Scenario: Server failure in a deployed release
- **WHEN** the server reports an unexpected failure from a deployed release
- **THEN** the Sentry event identifies that release

#### Scenario: Browser error in a deployed release
- **WHEN** the browser reports an error from a deployed release
- **THEN** the Sentry event identifies the same release as server events from that deployment

#### Scenario: Browser event contains a minified stack frame
- **WHEN** Sentry receives a browser event from a deployed release with a minified JavaScript stack frame
- **THEN** Sentry can resolve the frame to the corresponding authored source for that release

#### Scenario: Published static assets are inspected
- **WHEN** a browser release is prepared for deployment
- **THEN** no source-map file is included among the publicly served static assets

## MODIFIED Requirements

### Requirement: Server error monitoring
The serving runtime SHALL initialise the official Sentry Go SDK when a valid DSN is configured, SHALL label events with the configured WGA environment and deployed release, and SHALL use bounded flushing during shutdown. It SHALL capture unhandled server failures without changing the response or error-propagation behaviour observed by clients.

#### Scenario: Unhandled server error

- **WHEN** a request handler returns an unexpected server error while Sentry is configured
- **THEN** the error is reported to Sentry and the existing client response behaviour is preserved

#### Scenario: Server panic

- **WHEN** request processing panics while Sentry is configured
- **THEN** the panic is reported to Sentry and the application's existing panic handling remains in effect

#### Scenario: SDK initialisation fails

- **WHEN** Sentry cannot initialise from the configured DSN
- **THEN** the failure is logged and the application continues without Sentry monitoring

### Requirement: Browser error monitoring
The browser bundle SHALL include the official Sentry browser SDK and SHALL initialise it as the application starts when the shared layout supplies a non-empty DSN. The browser SDK SHALL use the configured WGA environment and deployed release and SHALL retain the application's existing browser initialisation behaviour.

#### Scenario: Browser receives a DSN

- **WHEN** a full page is rendered with a configured Sentry DSN
- **THEN** the browser initialises Sentry before the main application bootstrap completes

#### Scenario: Browser receives no DSN

- **WHEN** a full page is rendered with Sentry monitoring disabled
- **THEN** the browser skips Sentry initialisation and the main application bootstrap still completes

### Requirement: Browser event privacy
The browser SDK SHALL remove query strings and fragments from error-event, stack-frame, and breadcrumb URLs, remove both message and arguments from console breadcrumbs, and fail closed when a URL cannot be parsed, before sending events to Sentry.

#### Scenario: Page URL contains a sensitive query parameter

- **WHEN** a browser error occurs on a URL containing query parameters or a fragment
- **THEN** the event sent to Sentry contains the URL path without the query string or fragment

#### Scenario: Stack frame contains a sensitive query parameter

- **WHEN** a browser error stack frame contains a URL with query parameters or a fragment
- **THEN** the stack-frame URL sent to Sentry contains no query string or fragment

#### Scenario: Console breadcrumb contains form data

- **WHEN** a console breadcrumb contains logged form data
- **THEN** the event sent to Sentry excludes the breadcrumb message and arguments

#### Scenario: Malformed URL contains sensitive data

- **WHEN** a browser event URL cannot be parsed and contains query parameters or a fragment
- **THEN** the event sent to Sentry contains a fixed safe placeholder rather than the original URL

### Requirement: Public configuration boundary
The application SHALL expose to browser code only the public Sentry DSN, deployment environment, and deployed release required for browser monitoring. It SHALL NOT expose unrelated server configuration or secrets through the layout, static assets, or monitoring events.

#### Scenario: Rendered page configuration

- **WHEN** a page is rendered with Sentry configured
- **THEN** its browser-visible monitoring configuration contains the DSN, environment, and release only
