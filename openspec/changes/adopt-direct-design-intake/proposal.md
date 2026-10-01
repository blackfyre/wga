# Proposal

## Why

WGA's UI/UX is designed in the Claude Design project "WGA" (`b010aebe-b3bd-4543-927f-f890084b607a`). Its `WGA Prototype.dc.html` is the source of truth, and its `CHANGELOG.md` records every accepted design change, newest first, with the work a downstream build still has to do. The repository instead follows a manual process: a hand-copied standalone export embedded in the binary as `internal/assets/reference/visual-overhaul.html` (about 828 KB), served at the development-only `/tmp/visual-overhaul` routes, and two skills that refresh it and check parity against it. The copy drifts from the live design, enlarges every build, and leaves no record of which changelog entries the repository has implemented.

## What Changes

- Remove the embedded reference file, its `embed.FS`, the development-only `/tmp/visual-overhaul` and `/tmp/visual-overhaul/footer` routes, their boundary entries, and the tests and CodeQL exclusion that exist only for them.
- Re-point the Playwright footer-replacement coverage, which used the `/tmp/visual-overhaul/footer` fixture, at a real public page so the coverage survives.
- Add `docs/design-sync.md`, which records the design project, the authoritative file, the last implemented changelog entry, any entry in progress, and how to read the design.
- Add a repository skill, `design-intake`, that turns new changelog entries into proposed OpenSpec changes and covers parity checks against the live prototype.
- Retire the `update-visual-overhaul-preview` and `visual-overhaul-reference-parity` skills and update `AGENTS.md`.

## Non-Goals

- Implementing any design changelog entry. The 29 September 2026 entries are implemented separately in `sync-design-2026-09-29`.
- Changing public routes, rendered pages, or the visual system. The removed routes were never registered in production.
- Rewording existing specification requirements that cite the accepted visual-overhaul reference; they describe design provenance, not the removed route.

## Capabilities

### New Capabilities

### Modified Capabilities

None. No specification requirement names the removed development route or the embedded file, so this is process and tooling maintenance with no spec-level behaviour change (`skip_specs: true`).

## Impact

- `internal/assets/reference.go`, `internal/assets/reference/`, `internal/handlers/static/`, `internal/handlers/header_markup.go`, and their tests.
- `playwright-tests/preferences.spec.ts`, `playwright-tests/public-navigation.spec.ts`, and `.github/workflows/codeql.yml`.
- `docs/design-sync.md`, `AGENTS.md`, and `.agents/skills/`. The `.claude/skills/` entries are untracked local files and must be updated in each checkout.
- The release binary shrinks by roughly the size of the removed file.
