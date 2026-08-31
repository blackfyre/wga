# Tasks

- [x] 1. Add `GET /health` health handler and register it centrally. **Verification:** the route responds with HTTP 200 and `ok` without authentication.
- [x] 2. Exclude `/health` from trusted head-markup processing. **Verification:** middleware eligibility identifies `/health` as a technical boundary.
- [x] 3. Add focused route and middleware tests. **Verification:** `go test ./internal/handlers/health ./internal/handlers` passes.
