# Design

Create a dedicated `internal/handlers/health` route module and register it from the central handler registry. The route returns the constant `ok` response through PocketBase's request adapter, so it does not introduce persistence or configuration dependencies.

Add `/health` to the existing technical-boundary list used by the trusted-head-markup middleware. This preserves a shallow liveness check even when a caller sends an HTML `Accept` header.

Verification will build a disposable PocketBase router, request the route, and assert the response. Existing middleware eligibility coverage will assert that `/health` is a technical boundary.
