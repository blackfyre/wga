# Web Gallery of Art Documentation Alignment

## What This Is

This project is a maintenance-focused brownfield initiative for the existing Web Gallery of Art codebase. The immediate goal is to make contributor, planning, and operational documentation trustworthy again by aligning it with the repository's real structure, working commands, and current implementation patterns before further feature work continues.

The product being maintained is a server-rendered PocketBase application for browsing artists and artworks, searching the collection, using dual-mode comparisons, submitting feedback and guestbook entries, and sending postcards. This milestone is for maintainers and contributors who need the written guidance to match the code they are working in.

## Core Value

The repository's documentation must describe the code and workflows that actually exist, so maintainers can move quickly without being misled.

## Requirements

### Validated

- ✓ Contributors can run and extend a PocketBase-based Go application with Bun/PostCSS/Templ assets — existing
- ✓ Visitors can browse artists, artworks, static pages, inspiration, dual-mode views, feedback, guestbook, and postcards — existing
- ✓ The repo already contains a mapped current-state codebase reference under `.planning/codebase/` — existing

### Active

- [ ] Contributor-facing and planning documentation matches the current repository layout, commands, and generated-asset flow
- [ ] Obsolete or misleading guidance is corrected, removed, or clearly rewritten instead of preserved for legacy wording
- [ ] Small adjacent code or naming fixes are allowed when they are the cleanest way to eliminate documentation drift

### Out of Scope

- Large feature development unrelated to documentation accuracy — this milestone is maintenance-first before new feature work
- Re-architecting the application to match older docs — code correctness takes precedence over historical documentation
- Broad product redesign or content changes unrelated to repo/workflow correctness — not part of this cleanup pass

## Context

The application is a monolithic PocketBase-backed Go service rooted at `cmd/wga/main.go`, with feature handlers under `internal/handlers/`, templates under `internal/assets/templ/`, compiled assets under `internal/assets/public/`, migrations under `internal/migrations/`, and frontend source under `resources/`. Existing repo guidance still references older top-level paths such as `handlers/`, `utils/`, `assets/templ/`, and `migrations/`, which no longer reflect the actual layout.

The repository already has working development commands in `devenv.nix`, frontend tooling in `package.json`, and browser tests in `playwright-tests/`. The maintenance goal is to align instructions with those real commands and structures, and to prefer fixing small mismatches in code or naming where that is more durable than documenting around them.

## Constraints

- **Code Truth**: The current codebase and working commands are authoritative — documentation must adapt to them unless a small corrective code fix is clearly better
- **Brownfield**: Existing functionality must remain intact — this work is maintenance on a live codebase, not a reset
- **Scope Control**: Adjacent fixes are allowed only when they directly remove drift — avoid feature creep
- **Contributor Usability**: The result should help maintainers start the next phase of feature work without re-learning inaccurate paths or commands

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Treat this as a brownfield maintenance milestone | The repo already has substantial existing behavior and structure | — Pending |
| Prioritize documentation correctness over preserving old wording | Misleading instructions slow down future work more than doc churn does | — Pending |
| Let code take precedence over docs | The code and working commands are the source of truth for maintainers | — Pending |
| Allow small adjacent code fixes when they remove mismatches cleanly | Renaming or small corrections may be better than documenting around drift | — Pending |

---
*Last updated: 2026-03-16 after initialization*
