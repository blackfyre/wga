## Why

WGA's Go and frontend dependency graphs have available updates, including runtime dependencies and build tooling. Keeping the resolved graphs current reduces exposure to fixed upstream defects and keeps the application compatible with supported releases.

## What Changes

- Update the root Go and Bun dependency graphs to their latest supported releases, including required toolchain and configuration compatibility changes.
- Regenerate the tracked frontend lockfile and generated licence/SBOM artefacts from the final dependency graph.
- Refresh the required FOSSA evidence and Linux/amd64 build-selection review for upgraded `modernc.org/libc` and `modernc.org/sqlite`.

## Non-goals

- Changing application behaviour other than compatibility fixes required by upgraded dependencies.
- Adding a FOSSA exception, licence policy change, or legal conclusion.
- Updating ignored alternative JavaScript lockfiles.

## Risks and decisions

- PocketBase v0.40 requires Go 1.27, so the local and container build toolchains must move together.
- Updated `modernc` versions require fresh FOSSA evidence before the previous review can no longer be relied upon.
- Major frontend tooling updates can require configuration migration and browser binary refresh.

## End state

The root manifests and tracked lockfiles resolve current supported dependencies, the application builds and tests with the required toolchain, and current FOSSA and licence artefacts evidence the shipped graph.
