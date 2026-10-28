# Tasks

## 1. Holding location on selection plates

- [x] 1.1 Add an optional location to the plate projection and render it as a separate entry after the title and artist only when it is non-empty. Verify with a focused templ test covering a present and an absent location (`go test ./internal/assets/templ/components`).
- [x] 1.2 Resolve current-holding names for the works of a selection page and of each selection preview in one bounded lookup per grid, leaving the location empty when unset or unresolvable. Verify with focused handler tests (`go test ./internal/handlers/artists ./internal/artworks`).

## 2. Plate size and alignment

- [x] 2.1 Make the plate span three grid rows as a subgrid (record text, VIEW WORK, itinerary action), and let wrappers pass the rows through. Verify with `templ generate` and `bun run build`.
- [x] 2.2 Add one shared selection plate grid (two columns below `md`, fixed-width columns from `md`) and use it on the selection page and in the artist record's selection previews. Verify with `go test ./internal/assets/templ/pages`.
- [x] 2.3 Add Playwright geometry checks: equal image boxes and equal `VIEW WORK →` tops within a row, and equal plate size between a selection preview and its selection page (`mise run test:playwright --port 8092 playwright-tests/selection.spec.ts`).
