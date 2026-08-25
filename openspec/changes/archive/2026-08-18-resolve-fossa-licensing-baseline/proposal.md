## Why

FOSSA reports ten unresolved licence-policy flags for `modernc.org/sqlite` and `modernc.org/libc`. The exact FOSSA matches include Go files compiled into WGA's Linux/amd64 binary, so the findings cannot be resolved as unshipped-source noise without a legal determination.

## What Changes

- Record the exact FOSSA issue, licence-match, dependency-path, and build-selection evidence for the current `modernc` findings.
- Define WGA's FOSSA dependency inputs and exclude editor/tooling dependency metadata from application inventory.
- Align the README licence declaration and FOSSA badges with the repository licence and configured FOSSA project.
- Document the legal-review boundary for compiled copyleft findings and require a new decision before any FOSSA exception, policy change, CI enforcement, or driver replacement.
- Preserve existing FOSSA licence data, organisation-wide policy rules, and WGA's generated licence notices and CycloneDX SBOM.

## Capabilities

### New Capabilities
- `dependency-compliance-evidence`: Defines WGA's FOSSA dependency scope and the evidence required before resolving a compiled-dependency licensing finding.

### Modified Capabilities

- None.

## Impact

- FOSSA analysis configuration, README metadata, and WGA compliance documentation.
- Evidence for the current `modernc.org/sqlite@v1.54.0` and `modernc.org/libc@v1.74.1` findings.
- No FOSSA issue resolution, CI secret, CI policy gate, application runtime behaviour, public API, or organisation-wide policy change.
