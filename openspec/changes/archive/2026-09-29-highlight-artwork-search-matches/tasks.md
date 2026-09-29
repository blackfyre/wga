# Tasks

## 1. Shared highlighter

- [x] 1.1 Move `ArtistName`/`splitNameHighlight` from `internal/assets/templ/pages/artists.templ` into a shared `components.HighlightMatch` helper, switch `/artists` to it, and move the escaping, case-folding and Unicode tests with it. Verify with `templ generate`, `go test ./internal/assets/templ/components ./internal/assets/templ/pages`, and confirm the existing artists tests still pass unchanged in behaviour.

## 2. Artwork results highlighting

- [x] 2.1 Add `TitleHighlight` and `ArtistHighlight` to `ArtworkSearchResultsView` and resolve them in `buildArtworkSearchResults` using the precedence in design.md (`title` over `q` for the title; `artist` over `q` for the artist unless `artist_id` is set). Verify with a focused table test in `internal/handlers/artworks` (`go test ./internal/handlers/artworks -run '^TestArtworkSearchHighlightTerms$'`).
- [x] 2.2 Render the highlighted title and the separately rendered artist name, separator and date in the grid and list views of `internal/assets/templ/pages/artworks.templ`. Keep `alt` and `title` attributes plain and keep the `NOT RECORDED` fallback. Verify with `templ generate` and render tests in `internal/assets/templ/pages` covering a title match, an artist match, escaping (`<mark>&lt;b&gt;</mark>`), unchanged `alt` text and no `mark` without a text search.

## 3. Integration verification

- [x] 3.1 Extend the artwork search integration test to assert that `/artworks?q=<term>` and `/artworks/results?q=<term>` return `mark`-wrapped matches and that `artist_id` suppresses highlighting from the `artist` filter. Verify with `go test ./internal/handlers/artworks -run 'Search'`.
- [ ] 3.2 Run `go mod tidy`, `go vet ./...`, `go test ./... -cover` and `mise run check`, and confirm all pass.
  - Archived deliberately with this task open. `go mod tidy`, `go vet ./...` and `go test ./... -cover` passed. `mise run check` could not run its golangci-lint step: the local v2.12.2 binary is built with Go 1.26 and rejects the `go 1.27.1` module, and building it under Go 1.27.1 makes staticcheck panic while analysing the standard library. Lint remains unverified.
