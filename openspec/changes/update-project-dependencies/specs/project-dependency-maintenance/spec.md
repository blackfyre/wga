# Project Dependency Maintenance

## Requirements

### Requirement: Current supported dependency graphs
WGA SHALL resolve every root Go and Bun dependency to its latest supported release, subject to the module ecosystem's major-version compatibility rules.

#### Scenario: Dependency manifests are resolved
- **WHEN** Go module and Bun dependency resolution run from the repository root
- **THEN** `go.mod`, `go.sum`, `package.json`, and `bun.lock` describe the final resolved dependency graphs without manifest drift

### Requirement: Aligned build toolchains
WGA SHALL use the minimum Go version required by its resolved Go dependency graph consistently in local development and container builds.

#### Scenario: Application is built in supported environments
- **WHEN** the local and container build definitions are inspected after the update
- **THEN** they select a Go version accepted by the resolved PocketBase release

### Requirement: Refreshed compliance evidence
WGA SHALL replace version-specific FOSSA evidence when an upgraded `modernc.org/libc` or `modernc.org/sqlite` module changes the reviewed version, without treating the prior evidence as a legal conclusion.

#### Scenario: Reviewed runtime modules change version
- **WHEN** the final dependency graph contains new `modernc.org/libc` or `modernc.org/sqlite` versions
- **THEN** the repository records their current FOSSA match identifiers and Linux/amd64 CGO-disabled build selection, and does not add a FOSSA exception or policy approval

### Requirement: Compatible shipped artefacts
WGA SHALL regenerate its browser assets, licence notices, and CycloneDX SBOM from the final dependency graph.

#### Scenario: Build-owned artefacts are generated
- **WHEN** the application build runs after the dependency update
- **THEN** the browser bundle, notices, and SBOM represent the resolved frontend and Go dependencies
