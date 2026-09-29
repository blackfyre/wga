# Design

## Context

`/artists` highlights matches in the template: `ArtistName(name, query)` and `splitNameHighlight` in `internal/assets/templ/pages/artists.templ` escape every segment and wrap the first rune-wise, case-folded match in `<mark>`. Tests in `artists_test.go` cover escaping, case folding and Unicode length changes such as `İ`.

The artwork results views (`ArtworkSearchResults` and `ArtworkSearchResultList`) receive `ArtworkSearchResultsView`, which carries no search terms. The identity line is built as one string, `artworkSearchMetadata(FilingName, Date)`, so the artist name cannot be marked up on its own. The SQL filters (`filters.go`) match `q` against `title || author.filing_name`, `title` against `title`, and `artist` against `author.filing_name`.

## Goals / Non-Goals

**Goals:**

- One highlight renderer shared by `/artists` and `/artworks`.
- Highlight terms chosen by the server from the same filter state that produced the results.

**Non-Goals:**

- Aligning SQLite `LIKE` case folding with Go Unicode folding.
- Highlighting multiple occurrences of a term.

## Decisions

- **Move the highlighter into the shared `components` package.** `ArtistName` and `splitNameHighlight` become a shared `components.HighlightMatch(text, query)` plus its splitter. `/artists` calls the shared version, and the existing tests move with the helper. The alternative, a copy in `artworks.templ`, would leave two escaping-sensitive implementations that could drift apart.
- **The handler resolves terms into two fields.** `buildArtworkSearchResults` sets `TitleHighlight` and `ArtistHighlight` on `ArtworkSearchResultsView`:
  - title term = `title`, or `q` if `title` is empty;
  - artist term = `artist` when `artist_id` is empty, otherwise `q`.

  The terms are passed through exactly as the filters use them, so each highlight matches what the database searched for. The templates stay free of filter logic. The alternative, passing the raw filters into the view, would spread query rules into Templ.

- **Render the identity line in parts.** The grid and list views render the highlighted artist name, the `·` separator and the date as separate pieces. They keep the existing `NOT RECORDED` fallback when both are empty, so the visible text is unchanged.

## Risks / Trade-offs

- [SQLite `LIKE` folds only ASCII case, while the highlighter folds Unicode] → Every row the database returns still contains a match the highlighter can find, so the worst case is a result with no highlight. `/artists` already behaves this way.
- [`<mark>` uses the browser's default styling, which may look out of place in dark mode] → This is accepted to match `/artists`. Restyling it belongs in a separate change.
- [Bionic reading processes result titles] → `mark` is already on its skip list, so highlighted text is left intact.
