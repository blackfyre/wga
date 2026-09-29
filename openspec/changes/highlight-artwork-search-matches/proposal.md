# Proposal

## Why

On `/artists`, the part of each name that matches the search text is highlighted. `/artworks` shows its results without any highlight, so visitors cannot quickly see why a work matched their text search, whether through its title or its artist.

## What Changes

- Catalogue results in grid and list views highlight the first case-insensitive match of the active text search in the title and in the artist name:
  - the free-text query (`q`) highlights both the title and the artist name;
  - the `title` filter highlights the title;
  - the `artist` text filter highlights the artist name, but only when no exact `artist_id` filter is active.
- The artist-name part of the result metadata line is rendered separately from the date so that only the name can carry a highlight; the visible text stays the same.
- `/artists` and `/artworks` share one highlight renderer. The current `/artists` behaviour, including its output, escaping and Unicode handling, is kept.
- Non-goals:
  - highlighting technique (the result cards don't show it);
  - highlighting in the Dual Mode result picker;
  - highlighting inside attribute text (`alt`, `title`, `aria-label`);
  - highlighting more than one match per field;
  - changing search matching semantics;
  - adding new `<mark>` styling.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: adds a requirement that catalogue results visibly highlight the matched text search terms.

## Impact

- Templ: the artwork search results grid and list views in `internal/assets/templ/pages/artworks.templ`; the artist name highlight in `internal/assets/templ/pages/artists.templ`, which moves to a shared helper.
- Handler: the artwork search results view gains the active highlight terms (`internal/handlers/artworks`).
- No schema, configuration, routing, JavaScript or CSS changes. HTMX fragment targets and swaps are unchanged.
