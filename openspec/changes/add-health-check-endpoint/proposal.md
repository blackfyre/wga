# Add health check endpoint

## Why

The local application launcher already probes `/health`, but the application does not serve that route. A stable liveness endpoint is required for local process supervision and deployment health probes.

## Scope

- Add an unauthenticated `GET /health` endpoint.
- Return a successful plain-text response without reading application data.
- Exclude the endpoint from document-only head-markup processing.

## Non-goals

- Database readiness checks.
- Version, configuration, or diagnostic output.
