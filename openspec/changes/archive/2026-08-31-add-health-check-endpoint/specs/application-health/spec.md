## ADDED Requirements

### Requirement: Liveness endpoint

The application SHALL expose an unauthenticated `GET /health` endpoint for liveness probes. It SHALL return HTTP 200 with the plain-text body `ok` and SHALL not read application data.

#### Scenario: Process is live

- **WHEN** a client sends an unauthenticated `GET /health` request
- **THEN** the application responds with HTTP 200 and the body `ok`

### Requirement: Health endpoint bypasses document processing

The health endpoint SHALL be treated as a technical boundary and SHALL not invoke document-only trusted head-markup processing.

#### Scenario: Health probe advertises HTML

- **WHEN** a client sends `GET /health` with an `Accept: text/html` header
- **THEN** the request is excluded from trusted head-markup processing
