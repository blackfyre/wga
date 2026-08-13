## 1. Baseline schema

- [x] 1.1 Capture the current PocketBase collection definitions, indexes, and access rules as the baseline source of truth.
- [x] 1.2 Replace project-owned historical migrations with ordered initial-settings, current-schema, and synthetic-bootstrap migrations.
- [x] 1.3 Define every target collection from the design, including `Art_periods`, postcard delivery collections, contributor collections, and `users` rules.
- [x] 1.4 Exclude obsolete synthetic-source fields and collections from the new schema migration.
- [x] 1.5 Inventory all PocketBase lookups, queries, saves, and filters; verify their collection, field, relation, and index contracts against the baseline and update affected calls.
- [x] 1.6 Define `(published, name, id)` and `(published, title, id)` browse indexes and the `(status, available_at, id)` postcard claim index in the baseline schema.
- [x] 1.7 Name administrator-managed collections with leading-capital snake case, name tracking collections with lower snake case, and retain `users` unchanged.

## 2. Synthetic bootstrap

- [x] 2.1 Extend the embedded source loader to read and validate the `art_periods` taxonomy.
- [x] 2.2 Import art periods into PocketBase with stable identifiers, slugs, date ranges, and descriptions before dependent reference content.
- [x] 2.3 Preserve complete current artist, artwork, music, glossary, guestbook, strings, static-page, relation, and asset import transformations.
- [x] 2.4 Retain the empty-application guard and target-field, relation, asset, and file-size validation.

## 3. Verification and local reset

- [x] 3.1 Update migration tests to assert the clean target schema has no legacy synthetic-source fields or collections.
- [x] 3.2 Update synthetic bootstrap tests to assert all fixture content, including art periods, relations, and assets, imports to a fresh database.
- [x] 3.3 Verify the populated-database path does not modify existing records.
- [x] 3.4 Recreate local `wga_data`, run focused migration and seed tests, then run `go test ./... -cover`.
- [x] 3.5 Add or update focused tests for every database model call changed by the baseline audit.
- [x] 3.6 Replace artist and artwork pagination's unbounded total-record loads with count-only queries that preserve their filters.
- [x] 3.7 Batch sitemap artwork-author retrieval and verify no per-artwork relation expansion remains.
- [x] 3.8 Verify query plans use the baseline browse and postcard claim indexes with production-scale fixture data.
- [x] 3.9 Update raw SQL, collection lookups, relation definitions, and focused tests for the renamed tracking collections.
