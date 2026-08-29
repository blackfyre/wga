## 1. Dependency resolution
- [x] 1.1 Update the Go and Bun dependency graphs, their required toolchain pins, and incompatible tooling configuration. **Verification:** manifests and lockfile are resolved without drift; frontend build and Go compilation succeed.

## 2. Compliance evidence
- [x] 2.1 Regenerate licence notices and the SBOM, then record fresh FOSSA and Linux/amd64 build-selection evidence for upgraded modernc modules. **Verification:** generator tests pass, generated artefacts contain the final modules, and FOSSA analysis succeeds.

## 3. Compatibility verification
- [x] 3.1 Verify the final graph with focused and full Go checks, a CGO-disabled Linux/amd64 build using copied existing data, and a Playwright browser test with updated browser binaries. **Verification:** all commands pass.
