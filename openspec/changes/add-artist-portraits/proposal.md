## Why

Artist pages reserve a portrait panel, but the application neither models nor renders the biography images now available from the authoritative artist source. Showing those images makes the artist record, its social preview, and its structured metadata represent the same public identity.

## What Changes

- Add an optional portrait image field to artist records.
- Import the portrait filename from the source database's existing biography-image output path; asset transfer and validation remain outside this change.
- Render supplied portraits on artist pages while retaining the current portrait panel as a fallback for artists without one.
- Use a supplied portrait as the artist page's Open Graph image and `Person.image` JSON-LD value.

## Capabilities

### New Capabilities
- `artist-portraits`: Display and publish metadata for optional artist biography portraits.

### Modified Capabilities
- `database-baseline-bootstrap`: Extend the current PocketBase artist schema and source-fixture import contract with optional portrait filenames while remaining compatible with the existing image-free fixture.

## Impact

- PocketBase artist collection schema and migration tests.
- SQLite seed-source reader and artist record importer.
- Artist DTO, renderer, template, Open Graph metadata, and JSON-LD generation.
- No new external dependency, upload flow, asset-copying process, or existing-installation backfill.
