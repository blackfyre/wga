## Why

The 1 October design parity audit (items AS-1 to AS-4) found that artwork search no longer matches the Claude Design prototype. Its pagination still uses the old `Previous` / `Next Page` links, and an unavailable direction renders as an `<a>` element without `href`. The facet order differs from the design. The school and form facets show native checkboxes rather than the design's text rows. The empty-state copy differs.

## What Changes

- Replace the artwork-search pagination with the design's row: `PAGE n OF m` on the left and `← PREV` / `NEXT →` on the right. Available directions are ordinary links that work without JavaScript and are enhanced with HTMX. An unavailable direction is non-link text.
- Reorder the filter facets to follow the design: TITLE OR ARTIST, COLLECTION, SCHOOL, FORM, TYPE, then the repository's TECHNIQUE and PERIOD extras, then YEAR RANGE.
- Present SCHOOL and FORM options as text rows: the label on the left, and a selected marker and count on the right. Each row keeps a visually hidden native checkbox, so the filter form still submits without JavaScript. The server-side count order is unchanged.
- Replace the filtered empty-state copy with the design's "No works match these filters." heading and guidance.
- Remove the unused generic `internal/utils` pagination renderer.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: artwork-search paging, facet order and presentation, and the filtered empty state.

## Impact

- `internal/assets/templ/pages/artworks.templ`, `internal/handlers/artworks/` and `internal/utils/pagination.go`.
- The Playwright artwork-search specs and the Go template and handler tests.
- No data, API or HTMX endpoint changes. Pagination keeps its `/artworks/results` request and its `#artwork-search-results` swap target.
