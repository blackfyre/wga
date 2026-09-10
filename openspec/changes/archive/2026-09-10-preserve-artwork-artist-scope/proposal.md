## Why

Artwork search opened from an artist record is intended to remain constrained by that exact artist while visitors refine the catalogue, but the observed browser interaction can lose that scope and the page gives no visible indication that an exact artist filter is active. The interaction needs browser-level assurance and a clear artist-scope presentation without repurposing the free-text query.

## What Changes

- Preserve the exact `artist_id` constraint through browser-driven artwork-search refinements and the resulting canonical URL.
- Show the resolved public artist name as a distinct active scope on artwork search while keeping the title-or-artist query available for additional text refinement.
- Keep the exact artist identifier as non-visible request state; do not expose an editable artist-ID control or place the artist name in the query value.
- Retain the existing reset, unknown-identifier, legacy artist URL, sorting, paging, result-view, and Dual Mode semantics.
- Add browser coverage that enters artwork search from an artist record and proves both the visible scope and its preservation after refinement.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: Make the active exact-artist scope visible and require browser-driven refinements to retain that scope while leaving free-text refinement independent.

## Impact

- Artwork-search view construction and Templ presentation.
- Exact artist lookup through the existing public artist catalogue boundary.
- Artwork-search handler/template tests and Playwright coverage.
- No route, database schema, dependency, or public parameter changes.
