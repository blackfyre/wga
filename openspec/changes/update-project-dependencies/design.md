## Approach

Update the Go graph with Go module tooling and the frontend graph with Bun. Keep the Go, Docker, and release toolchains aligned with PocketBase's minimum Go version. Migrate only configuration made invalid by upgraded tooling.

After the final graph is resolved, regenerate build-owned licence artefacts, upload the configured FOSSA analysis, and record the resulting `modernc` findings and Linux/amd64 file selection without asserting a legal conclusion.

## Operational Decomposition

One owner updates manifests, compatibility configuration, evidence, and generated artefacts serially because each later step depends on the final dependency graph.

## Verification

- Confirm Go and Bun resolve the final manifests without drift.
- Run frontend build, template generation, focused licence tests, Go vet, and the Go test suite.
- Build a Linux/amd64, CGO-disabled binary and exercise it against a copied existing data directory.
- Run FOSSA target discovery and analysis for the final graph; record the refreshed findings and build selection.
- Run a Playwright browser test with the matching installed browser binary.
