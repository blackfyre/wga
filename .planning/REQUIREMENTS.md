# Requirements: Web Gallery of Art Documentation Alignment

**Defined:** 2026-03-16
**Core Value:** The repository's documentation must describe the code and workflows that actually exist, so maintainers can move quickly without being misled.

## v1 Requirements

### Documentation Audit

- [ ] **AUDT-01**: Maintainer can see a concrete inventory of outdated path, command, and workflow references across contributor-facing and planning documentation
- [ ] **AUDT-02**: Maintainer can trace each documented mismatch back to the current repo file, command, or structure that proves the correction

### Contributor Guidance

- [x] **DOCS-01**: Contributor can read repository guidance that uses the current directory structure, including `cmd/wga`, `internal/*`, `resources/*`, and `playwright-tests/*`
- [x] **DOCS-02**: Contributor can follow a canonical set of run, build, test, and generation commands that matches the repo's actual workflow definitions
- [x] **DOCS-03**: Contributor can tell which template and asset files are source files and which files are generated outputs that should be regenerated

### Planning and Internal Docs

- [ ] **PLAN-01**: Maintainer can read `.planning` documents whose brownfield context reflects the current PocketBase application's capabilities and this documentation-maintenance scope instead of historical assumptions
- [ ] **PLAN-02**: Maintainer can see the documented rule that code and working repo configuration take precedence over stale documentation when planning docs describe current commands, paths, and behavior

### Corrective Cleanup

- [ ] **CLNP-01**: Maintainer can remove or rewrite obsolete guidance rather than keeping misleading historical wording in active docs
- [ ] **CLNP-02**: Maintainer can make small adjacent code or naming fixes when they are the cleanest low-risk way to eliminate documentation drift

### Verification and Guardrails

- [ ] **VERI-01**: Maintainer can verify documentation changes against the repo's actual commands, file structure, and relevant configuration files before considering the cleanup complete
- [ ] **VERI-02**: Maintainer can leave behind lightweight guardrails that reduce the chance of path and command drift returning immediately after cleanup

## v2 Requirements

### Automation

- **AUTO-01**: Repository automatically checks documentation references against current command and path definitions in CI
- **AUTO-02**: Repository ships a reusable docs-maintenance checklist or linting workflow for future milestones

## Out of Scope

| Feature | Reason |
|---------|--------|
| New product features unrelated to documentation correctness | This milestone is maintenance-first before more feature work |
| Broad codebase re-architecture to satisfy old docs | Code and working repo configuration are the source of truth; docs should align to them |
| Full rewrite of every document in the repository | Too broad for the maintenance goal and likely to create unnecessary churn |
| Heavy automation for docs drift detection | Valuable later, but not required for the current cleanup pass |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUDT-01 | Phase 1 | Complete |
| AUDT-02 | Phase 1 | Complete |
| DOCS-01 | Phase 2 | Complete |
| DOCS-02 | Phase 2 | Complete |
| DOCS-03 | Phase 2 | Complete |
| PLAN-01 | Phase 3 | Complete |
| PLAN-02 | Phase 3 | Complete |
| CLNP-01 | Phase 4 | Pending |
| CLNP-02 | Phase 4 | Pending |
| VERI-01 | Phase 5 | Pending |
| VERI-02 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-16*
*Last updated: 2026-03-16 after Phase 1 execution*
