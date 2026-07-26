# Issue 187 Licence Notices and SBOM

## Introduction

Generate and verify the licence notices and CycloneDX SBOM for dependencies actually shipped in the WGA binary and browser bundle. The generated notices will be embedded and publicly available; the SBOM will be emitted with the build artefacts.

## Phase 1: Dependency data and generation

### Item: Build-owned dependency inventory

- [✓] Implement dependency discovery from `go list -deps -json` and the Bun metafile. **Verification:** every shipped Go module and browser package is identified with version, integrity data, target, and dependency edges.
- [✓] Add and validate the reviewed licence manifest. **Verification:** the build fails for missing, changed, or incomplete component records.
- [✓] Generate deterministic HTML notices and a CycloneDX 1.7 JSON SBOM. **Verification:** generated documents contain all reviewed components and valid dependency references.

## Phase 2: Application and delivery integration

### Item: Public notices and build artefacts

- [✓] Embed and render the public notices page, with a footer link. **Verification:** `/open-source-licences` returns the standard application layout and generated notices.
- [✓] Run generation during application and release builds. **Verification:** `dist/wga.cdx.json` is produced before compilation and included by release packaging.
- [✓] Document the regeneration and review workflow. **Verification:** contributor guidance names the source manifest, command, and release artefact.

## Phase 3: Verification and delivery

### Item: Automated assurance and pull request

- [✓] Add focused generator and route tests. **Verification:** tests cover discovery validation, deterministic output, CycloneDX structure, and the public route.
- [✓] Run focused and full quality checks. **Verification:** `go mod tidy`, `go vet ./...`, and `go test ./... -cover` complete successfully in that order.
- [✓] Commit and open a pull request. **Verification:** PR #191 targets `main`, references issue 187, and contains only this work.
