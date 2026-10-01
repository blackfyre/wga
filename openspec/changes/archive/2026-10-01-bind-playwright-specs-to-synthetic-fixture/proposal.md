# Proposal

## Why

Eight Playwright spec files address production records by ID (Gozzoli `r9fb82d431d2a5c`, "Kolowrat Wedding" `rea135c19d9c553`, Aachen "Boy with Grapes", production selections) or depend on palette and museum data. The Playwright runner and CI both seed the embedded synthetic dataset, so these 33 tests fail on every run. With `wga-src`'s `enrich-synthetic-fixture-records` providing palettes, public museums, and pinned scenario records, the specs can target synthetic records instead.

## What Changes

- Refresh `resources/synthetic/` by copying `wga-src/out/synthetic/` over it after the upstream change lands.
- Add one Playwright helper module that names each pinned synthetic scenario record (curated artist, its commentary and no-commentary selections, relationship artwork, palette artwork pair, a museum venue) with the WGA paths served for them, and a note of which specs depend on each.
- Rewrite `dual-production`, `record-production`, `release-inventory`, `artwork-record-relationships-task73`, `artwork-search-task71`, and `artwork-palette` specs to use the helper, keeping each test's behavioural intent. Assertions on production-only text (for example "GOZZOLI, Benozzo") become assertions on the synthetic record's own values.
- Rename "production" test titles where they no longer describe production data.

## Non-Goals

- Application behaviour changes. Spec-drift and product regressions found in the same run are handled by `repair-playwright-regressions`.
- Running any suite against the production seed.

## Capabilities

### New Capabilities

### Modified Capabilities

None; this is test-suite and fixture maintenance with no spec-level behaviour change (`skip_specs: true`).

## Impact

- `resources/synthetic/` (embedded fixture, binary), `playwright-tests/helpers/`, and the six spec files above.
- Go tests that assert synthetic fixture counts or contents (`internal/utils/seed`, `internal/migrations`, `internal/tours`) may need expectation updates after the copy.
- Depends on `wga-src` change `enrich-synthetic-fixture-records`.
