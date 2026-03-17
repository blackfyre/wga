# Roadmap: Web Gallery of Art Documentation Alignment

**Project:** Web Gallery of Art Documentation Alignment
**Generated:** 2026-03-16
**Mode:** yolo
**Granularity:** standard
**Parallelization:** false

## Overview

This roadmap restores trust in repository guidance before further feature work. It starts with a factual drift audit, then updates contributor and planning docs, allows narrow corrective cleanup where needed, and closes with verification and maintenance guardrails built around a source-of-truth workflow.

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Audit Drift Surface | Build a verified inventory of documentation mismatches against the current repo | AUDT-01, AUDT-02 | 4 |
| 2 | Correct Contributor Docs | Update high-traffic contributor/build/test docs to match real structure and commands | DOCS-01, DOCS-02, DOCS-03 | 4 |
| 3 | Align Planning Docs | Ensure `.planning` and internal maintenance guidance reflect the brownfield app accurately | PLAN-01, PLAN-02 | 4 |
| 4 | Remove Residual Mismatches | Rewrite or remove stale guidance and apply narrow corrective fixes where they simplify truth | CLNP-01, CLNP-02 | 4 |
| 5 | Verify and Guard | Confirm the cleanup against executable repo truth and add lightweight anti-drift guardrails | VERI-01, VERI-02 | 4 |

## Phase Details

### Phase 1: Audit Drift Surface

**Goal:** Build a verified inventory of documentation mismatches against the current repo.

**Requirements:** AUDT-01, AUDT-02

**Success criteria:**
1. A concrete list of outdated paths, commands, and workflow statements exists.
2. Each recorded mismatch is backed by a current repo file, config, or command reference.
3. High-traffic docs and planning docs are both included in the audit scope.
4. The audit separates factual mismatches from optional style improvements.

### Phase 2: Correct Contributor Docs

**Goal:** Update high-traffic contributor/build/test docs to match real structure and commands.

**Requirements:** DOCS-01, DOCS-02, DOCS-03

**Success criteria:**
1. Contributor-facing docs use current repo paths consistently.
2. Build, run, test, and generation instructions match `devenv.nix`, `package.json`, Go commands, and Playwright config where relevant.
3. Docs explain source-versus-generated boundaries for templates and built assets.
4. A contributor can follow the primary documented workflow without hitting stale references.

### Phase 3: Align Planning Docs

**Goal:** Ensure `.planning` and internal maintenance guidance reflect the brownfield app accurately.

**Requirements:** PLAN-01, PLAN-02

**Plans:** 2/2 plans complete

**Success criteria:**
1. `.planning` context documents describe the current application and maintenance scope accurately.
2. Brownfield validated capabilities remain distinct from current maintenance goals.
3. Source-of-truth rules are stated clearly in planning materials.
4. Planning docs no longer repeat historical path or workflow assumptions contradicted by the repo.

### Phase 4: Remove Residual Mismatches

**Goal:** Rewrite or remove stale guidance and apply narrow corrective fixes where they simplify truth.

**Requirements:** CLNP-01, CLNP-02

**Plans:** 2/2 plans complete

**Success criteria:**
1. Misleading active guidance is removed or rewritten rather than preserved as legacy wording.
2. Small adjacent fixes are applied only where they directly eliminate a recurring mismatch.
3. Cleanup stays within maintenance scope and does not turn into unrelated feature work.
4. Remaining exceptions, if any, are explicit and justified.

### Phase 5: Verify and Guard

**Goal:** Confirm the cleanup against executable repo truth and add lightweight anti-drift guardrails.

**Requirements:** VERI-01, VERI-02

**Plans:** 2/2 plans complete

**Success criteria:**
1. Updated docs are checked against the current repo structure, commands, and configuration files.
2. Any changed workflows are validated with appropriate spot checks or existing test/build commands where needed.
3. Maintainers are left with clear guidance on how to resolve future code-vs-doc conflicts.
4. The cleanup ends with a small, repeatable anti-drift checklist or equivalent guidance.

## Phase Ordering Rationale

- Phase 1 comes first because brownfield documentation cleanup should begin with facts, not rewriting.
- Phase 2 targets the highest-traffic contributor paths next so future work benefits immediately.
- Phase 3 aligns the planning layer to the corrected contributor baseline from Phase 2 and makes the source-of-truth rule explicit instead of reopening broad repo discovery.
- Phase 4 reserves space for targeted residual cleanup without contaminating the earlier factual phases.
- Phase 5 closes the loop so the maintenance pass is verified and less likely to regress.

## Progress

| Phase | Status | Plans | Progress |
|-------|--------|-------|----------|
| 1 | ✓ | 2/2 | 100% |
| 2 | ✓ | 2/2 | 100% |
| 3 | ✓ | 2/2 | 100% |
| 4 | ✓ | 2/2 | 100% |
| 5 | ✓ | 2/2 | 100% |

---
*Roadmap created: 2026-03-16*
*All v1 requirements mapped: yes*
