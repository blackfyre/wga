# Tasks

## 1. Fixture refresh

- [x] 1.1 After `wga-src`'s `enrich-synthetic-fixture-records` is applied and rebuilt, copy `../wga-src/out/synthetic/{wga-test.sqlite,storage}` over `resources/synthetic/`, then update any Go expectation that asserted the old fixture contents; verify with `go test ./internal/utils/seed/... ./internal/migrations/... ./internal/tours/...` and then `go test ./... -cover`.

## 2. Scenario helper

- [x] 2.1 Add `playwright-tests/helpers/synthetic-fixture.ts` exporting the pinned scenario paths and IDs exactly as WGA serves them after seeding (confirm each path returns HTTP 200 from a fresh `mise run test:playwright` server), with a comment per record naming the specs that depend on it; verify with `bunx biome check playwright-tests/helpers`.

## 3. Spec rewrites

- [x] 3.1 Rewrite `record-production.spec.ts` and `release-inventory.spec.ts` against the curated artist, selections, and relationship artwork; verify with `mise run test:playwright playwright-tests/record-production.spec.ts playwright-tests/release-inventory.spec.ts`.
- [x] 3.2 Rewrite `dual-production.spec.ts` against the curated artist and its selections; verify with `mise run test:playwright playwright-tests/dual-production.spec.ts`.
- [x] 3.3 Rewrite `artwork-record-relationships-task73.spec.ts` to open the relationship artwork directly (or by exact synthetic title search) and assert all three bases; verify with `mise run test:playwright playwright-tests/artwork-record-relationships-task73.spec.ts`.
- [x] 3.4 Update `artwork-search-task71.spec.ts` to use synthetic museum venues and synthetic year ranges; verify with `mise run test:playwright playwright-tests/artwork-search-task71.spec.ts`.
- [x] 3.5 Rewrite `artwork-palette.spec.ts` against the palette artwork pair, including the two-pane Dual URL; verify with `mise run test:playwright playwright-tests/artwork-palette.spec.ts`.

## 4. Integration

- [x] 4.1 Run the full `mise run test:playwright` and confirm none of the tests in the rewritten files fail; record any remaining failures as belonging to `repair-playwright-regressions`.
