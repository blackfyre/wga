## ADDED Requirements

### Requirement: Application-owned FOSSA dependency scope
The repository SHALL configure FOSSA to analyse WGA's root Go and frontend dependency manifests while excluding dependency manifests under `.opencode` from application inventory.

#### Scenario: Tooling manifest is excluded
- **WHEN** FOSSA lists analysis targets from the repository root
- **THEN** `.opencode/package-lock.json` is not an analysis target

#### Scenario: Runtime dependency manifests remain analysed
- **WHEN** FOSSA lists analysis targets from the repository root
- **THEN** the root Go module and frontend dependency target remain analysis targets

### Requirement: Canonical licence and FOSSA project metadata
The README SHALL declare Apache License 2.0, matching `LICENSE.md` and `package.json`. The FOSSA badges in the README and `.fossa.yml` SHALL reference the same configured project locator.

#### Scenario: Repository metadata is inspected
- **WHEN** a contributor compares the README, repository licence, package metadata, and FOSSA configuration
- **THEN** the Apache-2.0 declaration and FOSSA project locator are consistent

### Requirement: Compiled licensing evidence
WGA SHALL record each current `modernc.org/libc@v1.74.1` and `modernc.org/sqlite@v1.54.0` FOSSA issue ID, licence match, dependency origin, and Linux/amd64 Go build selection. The record SHALL identify whether each matched file is compiled by WGA.

#### Scenario: Current modernc findings are recorded
- **WHEN** the current `modernc.org/libc@v1.74.1` and `modernc.org/sqlite@v1.54.0` findings are reviewed
- **THEN** each finding has recorded FOSSA match and build-selection evidence

#### Scenario: A module version changes
- **WHEN** WGA upgrades either reviewed `modernc` module to another version
- **THEN** the earlier evidence record is not used as the legal basis for the new version and a fresh review is required

### Requirement: Legal-review boundary
The repository SHALL preserve the current FOSSA findings as unresolved and SHALL NOT add a FOSSA issue exception, organisation-wide policy approval, licence-data correction, or credentialed CI policy gate without a subsequent documented legal conclusion.

#### Scenario: No legal conclusion exists
- **WHEN** the compiled `modernc` licence evidence has been recorded without a legal conclusion
- **THEN** the FOSSA findings remain unresolved and no FOSSA policy gate is added

### Requirement: Shipped compliance artefact consistency
The change SHALL verify that WGA's generated licence notices and CycloneDX SBOM retain entries for the reviewed runtime modules.

#### Scenario: Regenerated compliance artefacts
- **WHEN** the licence generator runs after the FOSSA scope configuration is added
- **THEN** the generated notices and CycloneDX SBOM retain the reviewed `modernc` runtime module entries
