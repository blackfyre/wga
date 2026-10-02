## 1. Pagination

- [x] 1.1 Replace the artwork-search pagination with the design's `PAGE n OF m · ← PREV · NEXT →` row: real links enhanced with HTMX, and an unavailable direction as non-link text. Remove the unused `internal/utils` renderer. Update the Go tests.

## 2. Facets

- [x] 2.1 Reorder the facets to TITLE OR ARTIST, COLLECTION, SCHOOL, FORM, TYPE, TECHNIQUE, PERIOD, YEAR RANGE.
- [x] 2.2 Render SCHOOL and FORM options as text rows with a visually hidden native checkbox, a selected marker and a count. Update the Go tests.

## 3. Empty state

- [x] 3.1 Use the design's filtered empty-state copy. Update the Go tests.

## 4. Verification

- [x] 4.1 Add Playwright assertions at 390, 834 and 1440px for the pagination markup, HTMX and no-JavaScript navigation, the facet order, the facet rows and their count order, and the empty-state copy.
- [x] 4.2 Run `go vet ./...`, `go test ./... -cover`, `mise run check`, the focused Playwright specs and the full Playwright suite.
