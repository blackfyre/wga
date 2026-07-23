# Synthetic Migration Import

## Introduction

Replace the source-only synthetic schema with a one-time migration that imports
the embedded synthetic dataset into the existing PocketBase collections defined
by `tmp/db.dbml`. The migration must include its image and audio assets so a
fresh container deployment needs no separate seed command.

## Phase 1 — Import design

### Existing-schema mapping

- [✓] Confirm the authoritative target collections and identify source tables without a target.
- [✓] Replace source-only collection and field writes with mappings to the existing collections. **Verification:** a fresh migrated database has no extra collections or fields.
- [✓] Preserve source record IDs and target relations. **Verification:** artists, artworks, taxonomy, and music relations resolve by their expected IDs.
- [✓] Preserve the legacy migration filename and remove its already-applied source schema in the next migration. **Verification:** an upgraded populated database keeps its application records while source-only collections and fields are removed.

## Phase 2 — Migration bootstrap

### Embedded import

- [✓] Run the embedded SQLite and asset import from the pending migration. **Verification:** migration tests import the complete dataset without `seed:data`; PocketBase applies pending migrations before `serve` listens.
- [✓] Attach artwork and music files through PocketBase file fields. **Verification:** every imported artwork image and music source exists in the configured filesystem.
- [✓] Skip a populated non-system application database and refuse destructive rollback. **Verification:** pre-existing application data remains unchanged when the migration completes.

## Phase 3 — Validation

### Automated coverage

- [✓] Replace obsolete source-schema and manual-seed tests with migration mapping tests. **Verification:** targeted migration tests cover records, relations, files, repeat startup, and populated targets.
- [✓] Run backend validation. **Verification:** `go vet ./...` and `go test ./... -cover` pass.
- [ ] Manual check by the user that a fresh deployment displays the seeded data and assets correctly.
