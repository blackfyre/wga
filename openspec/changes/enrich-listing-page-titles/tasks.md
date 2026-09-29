# Tasks

## 1. Shared title fitting

- [x] 1.1 Add the pure title-fitting function and its part and reduction-step types to `internal/assets/templ/utils`. It covers long and short forms, lead, page-name and filter roles, `+N` aggregation of removed filters, lead truncation with `…`, and an 80 code-point budget that includes ` - WGA`. Verify with table tests for: no reduction under budget, each reduction step in order, removed filters counted, page name never removed, multi-byte characters counted as code points, and the final guard fitting an oversized lead (`go test ./internal/assets/templ/utils -run '^TestFitTitle'`).

## 2. Artist index

- [x] 2.1 Build artist-index title parts in `internal/handlers/artists` from name query, letter, school label, period label, narrowed birth range and page position, in the design's reduction order, and pass the result to `TitleKey` in `processArtists`. Verify with a table test of state to title, covering the unfiltered `Artists` title, first-page omission and `p. N/M` (`go test ./internal/handlers/artists -run '^TestArtistIndexTitle'`).

## 3. Artwork search

- [x] 3.1 Build artwork-search title parts in `internal/handlers/artworks` from artist scope, free-text query, title, artist-text and technique filters, school, form, type, period, collection and year selections, and page position. Derive the page count from the result count and `artworkSearchPageSize`, use normal-case labels, and exclude view and sort. Pass the result to `TitleKey`. Verify with a table test, including the spec example `“madonna” · Artworks Search · Florentine · p. 3/41` and a long artist scope (`go test ./internal/handlers/artworks -run '^TestArtworkSearchTitle'`).
- [x] 3.2 Emit a root-level `<title>` for the artwork-search block response and the `/artworks/results` fragment response without placing it inside `ArtworkSearchResults`, and verify that the full page has exactly one `<title>`, inside `<head>`. Verify with render tests asserting the root-level title on both HTMX responses and no body `<title>` on the full page, plus escaping of `q=<b>` (`templ generate && go test ./internal/handlers/artworks ./internal/assets/templ/pages`).

## 4. Global search

- [x] 4.1 Build the global-search title from the quoted term in `internal/handlers/search`, and emit a root-level `<title>` for the `/search/results` fragment without placing it inside `SearchResults`. Verify with tests for the full page and fragment titles and for the unfiltered `Search` title (`templ generate && go test ./internal/handlers/search ./internal/assets/templ/pages`).

## 5. Browser verification

- [x] 5.1 Add Playwright assertions that `document.title` updates after an artwork-search facet swap, and matches a full load of the pushed URL, stays unchanged on a view or sort switch, updates after an artist-index letter swap, and updates after debounced global-search typing. Verify with `mise run test:playwright -- artwork-search.spec.ts artists.spec.ts global-search.spec.ts`.
- [x] 5.2 Run the final gates: `go mod tidy`, `go vet ./...`, `go test ./... -cover` and `mise run check`. Report any lint step blocked by toolchain skew as blocked.
