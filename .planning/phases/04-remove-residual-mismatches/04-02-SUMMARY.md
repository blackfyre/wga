---
phase: 04-remove-residual-mismatches
plan: 02
subsystem: docs
tags: [planning, docs, playwright, mailhog, mailpit]
requires:
  - phase: 04-remove-residual-mismatches
    provides: updated README and secondary repo-analysis docs aligned to current workflow truth
provides:
  - Internal codebase reference docs aligned to the corrected contributor baseline
  - Playwright configuration comments updated to the current dist/wga binary path
affects: [phase-05-verification]
tech-stack:
  added: []
  patterns: [internal reference docs mirror active contributor guidance, config comments must match current binary paths]
key-files:
  created: [.planning/phases/04-remove-residual-mismatches/04-02-SUMMARY.md]
  modified: [.planning/codebase/CONVENTIONS.md, .planning/codebase/INTEGRATIONS.md, .planning/codebase/TESTING.md, playwright.config.ts]
key-decisions:
  - "Describe residual drift in internal reference docs as a regression risk now that Phase 2 corrected contributor-facing paths."
  - "Keep the local service name as mailhog while preserving MAILPIT_URL as the test-facing UI endpoint variable."
  - "Limit the adjacent code cleanup to the commented Playwright webServer example so runtime behavior remains unchanged."
patterns-established:
  - "Internal planning docs should reinforce the same dist/wga and mailhog or MAILPIT_URL model as README."
  - "Comment-only config fixes are acceptable when they remove stale guidance without changing behavior."
requirements-completed: [CLNP-01, CLNP-02]
duration: 8 min
completed: 2026-03-17
---

# Phase 4 Plan 02: Remove Residual Mismatches Summary

**Internal codebase references and the Playwright server example now reinforce the same corrected dist/wga and mailhog or MAILPIT_URL guidance as the active docs**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-17T06:55:38Z
- **Completed:** 2026-03-17T07:03:38Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Replaced the stale “repo instructions still use root-level paths” note in `.planning/codebase/CONVENTIONS.md` with the post-Phase-2 regression risk framing.
- Clarified in `.planning/codebase/INTEGRATIONS.md` and `.planning/codebase/TESTING.md` that local email capture comes from the `mailhog` service while Playwright inspects messages through `MAILPIT_URL`.
- Updated the commented Playwright `webServer.command` example to `./dist/wga serve --dev` without changing runtime configuration.

## Task Commits

Each task was committed atomically:

1. **Task 1: Align internal reference docs to the corrected contributor baseline** - `d1fcb32a` (docs)
2. **Task 2: Correct the stale Playwright webServer command comment** - `6eafd696` (docs)

## Files Created/Modified

- `.planning/codebase/CONVENTIONS.md` - reframed residual inconsistency as a future regression risk instead of a current contributor-doc fact
- `.planning/codebase/INTEGRATIONS.md` - clarified the `mailhog` service versus `MAILPIT_URL` UI split
- `.planning/codebase/TESTING.md` - aligned postcard end-to-end mail inspection wording to the same mail-capture model
- `playwright.config.ts` - fixed the commented built-binary example to use `./dist/wga serve --dev`
- `.planning/phases/04-remove-residual-mismatches/04-02-SUMMARY.md` - captured plan outcome and execution metadata

## Decisions Made

- Describe residual drift in internal reference docs as a regression risk now that Phase 2 corrected contributor-facing paths.
- Keep the local service name as `mailhog` while preserving `MAILPIT_URL` as the test-facing UI endpoint variable.
- Limit the adjacent code cleanup to the commented Playwright `webServer` example so runtime behavior remains unchanged.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 4 is complete. Ready for Phase 5 verification and guardrail work against the corrected contributor, planning, and secondary reference baseline.

## Self-Check: PASSED

---
*Phase: 04-remove-residual-mismatches*
*Completed: 2026-03-17*
