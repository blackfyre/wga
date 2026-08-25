## 1. Compliance evidence

- [x] 1.1 Record the FOSSA File Matches, licence evidence, and issue IDs for all current `modernc.org/libc@v1.74.1` and `modernc.org/sqlite@v1.54.0` findings.
- [x] 1.2 Record the Linux/amd64 release and Railway UAT build targets separately, including any architecture evidence or outstanding architecture confirmation.

## 2. Analysis scope and documentation

- [x] 2.1 Add a version-3 `.fossa.yml` that excludes `.opencode` while retaining WGA's root Go and frontend dependency targets.
- [x] 2.2 Use `fossa list-targets` to identify candidate targets, then run `fossa analyze --output` to verify that only the tooling target is removed from the final analysis.
- [x] 2.3 Document WGA's FOSSA scope, compiled-source evidence, legal-review boundary, reviewed module versions, and dependency-upgrade review rule.
- [x] 2.4 Align the README licence declaration and FOSSA badges with the repository licence and configured FOSSA project locator.

## 3. Shipped compliance artefacts

- [x] 3.1 Regenerate WGA's licence notices and CycloneDX SBOM, then verify the reviewed runtime modules remain represented.

## 4. Verification

- [x] 4.1 Run the focused FOSSA target and policy commands locally with the final configuration, confirming that the current findings remain unresolved.
- [x] 4.2 Run the existing licence-generator test and regenerate the committed notices/SBOM to confirm repository checks remain green.
- [x] 4.3 Perform a manual review of the FOSSA evidence, legal-review boundary, documentation, and generated attribution artefacts.
