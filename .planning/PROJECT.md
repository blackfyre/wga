# Web Gallery of Art Documentation Alignment

## What This Is

This project now has a shipped documentation-alignment baseline for the existing Web Gallery of Art codebase. The completed v1.0 milestone restored trustworthy contributor, planning, and operational guidance so future work can start from the repo's real structure, working commands, and current implementation patterns instead of stale prose.

The product being maintained is a server-rendered PocketBase application for browsing artists and artworks, searching the collection, using dual-mode comparisons, submitting feedback and guestbook entries, and sending postcards. The next milestone can build on this cleaner maintainer baseline instead of spending more time reconciling path and command drift.

## Current State

- Shipped milestone: **v1.0 Documentation Alignment** on `2026-03-17`
- Milestone outcome: contributor docs, planning docs, and residual cleanup are independently verified and tied together by a passed milestone audit
- Current planning baseline:
  - `cmd/wga/main.go`, `internal/*`, `resources/*`, and `playwright-tests/*` are the documented repo layout
  - `devenv.nix`, `package.json`, `.env.example`, and `playwright.config.ts` remain the canonical workflow evidence files
  - `docs/documentation-maintenance.md` is the standing anti-drift guardrail for future documentation edits

## Core Value

The repository's documentation must describe the code and workflows that actually exist, so maintainers can move quickly without being misled.

## Requirements

### Validated

- ✓ Contributors can run and extend a PocketBase-based Go application with Bun/PostCSS/Templ assets — existing
- ✓ Visitors can browse artists, artworks, static pages, inspiration, dual-mode views, feedback, guestbook, and postcards — existing
- ✓ The repo already contains a mapped current-state codebase reference under `.planning/codebase/` — existing
- ✓ Documentation audit and traceability coverage are complete and internally consistent — v1.0
- ✓ Contributor-facing docs match the current repo layout, command surface, and generated-asset flow — v1.0
- ✓ Planning docs follow the current brownfield context and explicit code-over-prose source-of-truth rule — v1.0
- ✓ Residual misleading documentation was cleaned up and bounded adjacent fixes were independently verified — v1.0
- ✓ Repo-truth verification and the documentation-maintenance guardrail close the loop against immediate drift — v1.0

### Active

- [ ] `AUTO-01`: Automatically check documentation references against current command and path definitions in CI
- [ ] `AUTO-02`: Ship a reusable docs-maintenance checklist or linting workflow for future milestones
- [ ] Define the next milestone's non-documentation goals now that the maintainer baseline is trustworthy again

### Out of Scope

- Reintroducing historical path or command wording that contradicts the shipped v1.0 baseline
- Large feature commitments before the next milestone requirements are explicitly defined
- Broad product redesign or content changes without a fresh milestone scope and roadmap

## Context

The application is a monolithic PocketBase-backed Go service rooted at `cmd/wga/main.go`, with feature handlers under `internal/handlers/`, templates under `internal/assets/templ/`, compiled assets under `internal/assets/public/`, migrations under `internal/migrations/`, and frontend source under `resources/`. That structure is now consistently reflected across contributor docs, planning docs, codebase reference docs, and the milestone archive.

The repository already has working development commands in `devenv.nix`, frontend tooling in `package.json`, and browser tests configured through `playwright.config.ts`. The v1.0 milestone used those files as the source of truth, produced 9 completed phases, 15 completed plans, and 30 task-level commits, and left the repo with a passed milestone audit plus a checklist-based anti-drift maintenance loop.

## Constraints

- **Code Truth**: The current codebase and working commands are authoritative — documentation must adapt to them unless a small corrective code fix is clearly better
- **Brownfield**: Existing functionality must remain intact — this work is maintenance on a live codebase, not a reset
- **Scope Control**: Adjacent fixes are allowed only when they directly remove drift — avoid feature creep
- **Contributor Usability**: The result should help maintainers start the next phase of feature work without re-learning inaccurate paths or commands

## Source of Truth

The rule is simple: code and working repo configuration override stale prose. Planning and contributor docs should be updated against the current implementation rather than preserving historical wording that no longer matches the repository.

Canonical evidence files for documentation decisions:
- `cmd/wga/main.go`
- `internal/*`
- `devenv.nix`
- `package.json`
- `playwright.config.ts`

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Treat this as a brownfield maintenance milestone | The repo already has substantial existing behavior and structure | ✓ Good |
| Prioritize documentation correctness over preserving old wording | Misleading instructions slow down future work more than doc churn does | ✓ Good |
| Let code take precedence over docs | The code and working commands are the source of truth for maintainers | ✓ Good |
| Allow small adjacent code fixes when they remove mismatches cleanly | Renaming or small corrections may be better than documenting around drift | ✓ Good |
| Add retro-verification phases when summary claims are not enough for milestone proof | Independent verification artifacts are required for trustworthy milestone closeout | ✓ Good |
| Preserve the guardrail as a checklist before investing in automation | The repo needed an immediate lightweight control before CI-based docs automation | ✓ Good |

## Next Milestone Goals

- Decide whether the next milestone should focus on documentation automation (`AUTO-01`, `AUTO-02`) or return to product work with the cleaned maintainer baseline.
- Start from a fresh requirements definition instead of extending the archived v1.0 scope.
- Keep the documentation-maintenance checklist and code/config source-of-truth rule in force for all future planning.

---
*Last updated: 2026-03-17 after v1.0 milestone completion*
