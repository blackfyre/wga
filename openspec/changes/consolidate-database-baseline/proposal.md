## Why

WGA's fresh-database path applies 21 application migrations, including obsolete JSON seeders and a synthetic-source migration that first adds fields and collections and then removes them. The versioned synthetic SQLite database is now the authoritative bootstrap input, and no production migration history must be preserved.

## What Changes

- **BREAKING** Replace the historical application migration chain with a concise baseline that creates only the current PocketBase collections, fields, indexes, rules, and configuration.
- Replace historical JSON and reference-file seeders with one synthetic bootstrap migration that imports the embedded SQLite fixture and its storage assets.
- Retain and seed `Art_periods` from the synthetic fixture so it is ready for the planned search filter.
- Preserve every current feature and its complete synthetic bootstrap content, including records, relations, and assets.
- Do not create source-only fields or collections in the PocketBase target schema.
- Preserve the synthetic fixture as a normalised, embedded source database rather than treating it as a PocketBase runtime database.
- Verify every database model call against the baseline schema and update calls that rely on removed or changed collections or fields.
- Name PocketBase collections administered through the UI with leading-capital snake case; name WGA tracking collections lower snake case, while retaining `users` unchanged.
- Add query-path indexes and count behaviour suitable for the expected 5,000 artists and 56,000 artworks.
- Add fresh-database verification for target schema, record counts, relations, and embedded assets.

## Capabilities

### New Capabilities

- `database-baseline-bootstrap`: A fresh WGA database receives the complete current PocketBase schema and synthetic reference content from a concise migration baseline.

### Modified Capabilities

None.

## Impact

- Affects `internal/migrations/`, `internal/utils/seed/`, `resources/synthetic/`, and all packages that query PocketBase collections.
- Replaces the current development-only database history; existing local data directories must be recreated.
- Does not change public HTTP routes or add dependencies.
