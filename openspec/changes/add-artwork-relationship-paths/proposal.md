## Why

Artwork records currently expose a generic `author` relation and free-form location and technique text. This cannot distinguish authorship qualities or reliably surface why two artworks are related.

The collection needs explicit, curator-managed relationships that support related-artwork discovery without publishing ambiguous private-collection data.

## What Changes

- **BREAKING** Replace the existing artwork `author` relation with an explicit primary-author relation and fixed additional authorship-quality relations.
- Add curated artwork relations for series/altarpieces, locations, museums, subjects, techniques, and art periods; continue using the existing school relationship for school/workshop connections where applicable.
- Derive public related-artwork paths from shared relationship values and label each path with its reason.
- Preserve source location text while classifying private collections internally and resolving current museums through canonical records and reviewed aliases.
- Provide an iterative unresolved-location report so new location variations can be reviewed and mapped without unsafe automatic links.

## Capabilities

### New Capabilities

- `artwork-relationship-paths`: Maintains explicit artwork relationships and presents related artworks with their shared connection paths.
- `location-normalisation`: Classifies raw artwork locations conservatively and maps verified museum variants to canonical museum records.

### Modified Capabilities

- None.

## Impact

- PocketBase artwork schema and data migrations.
- Synthetic-data import and the source-to-application import contract.
- Artwork search, artwork detail, artist detail, URL generation, JSON-LD, and related-artwork presentation.
- Curatorial administration in the PocketBase dashboard.
