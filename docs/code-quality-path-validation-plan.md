# Code Quality Path Validation Plan

## Introduction

This delivery adds focused tests for the synthetic-seed path boundary. Music paths from the embedded SQLite source are normalised and rejected when they could escape the embedded storage root, but this contract had no direct test coverage.

## Phase 1: Assess the existing boundary

### Current behaviour and constraints

- [✓] Trace the music-file loading path. Verification: `loadSourceFiles` passes each `LocalPath` through `sourceMusicFilePath` and `safeRelativePath` before `io/fs` reads it.
- [✓] Confirm the smallest useful scope. Verification: valid nested paths and empty, absolute, direct traversal, and normalised traversal paths are covered without changing production code or migration behaviour.

## Phase 2: Lock down the contract

### Focused unit coverage

- [✓] Add table-driven `safeRelativePath` tests. Verification: valid paths return their cleaned relative path; empty, absolute, and parent-traversal paths return an error.
- [✓] Run the focused Go test. Verification: `go test ./internal/utils/seed -run '^TestSafeRelativePath$'` passes.
- [ ] Manually review the scope. Verification: only the path-boundary test and this delivery plan changed.
