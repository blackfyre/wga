# Proposal

## Why

The artist index (`/artists`), artwork search (`/artworks`) and global search (`/search`) render one fixed document title, such as `Artworks Search - WGA`, whatever query, filters or page is shown. Tabs, history entries and screen-reader page announcements therefore cannot tell one search state from another. Artwork search swaps also push a new URL while leaving the title unchanged.

## What Changes

- Build the document title of these three listing pages from their active state. This includes the free-text query, artist scope, letter, facet selections and page position.
- Give each title part a long and a short form. Fit the title within a fixed maximum length by shortening, then dropping, parts in a defined priority order. The page name and the ` - WGA` suffix are never dropped.
- Write the page position as `p. N/M` and omit it on the first page.
- Update the title on HTMX swaps of these pages as well as on full page loads. That includes the artwork-search block and result fragment and the global-search result fragment, which today carry no `<title>`.
- Keep link-preview titles (`og:title`, `twitter:title`) equal to the document title, as today.

## Non-goals

- Result counts in titles.
- Search-engine title optimisation, or a canonical or robots policy for faceted URLs.
- Titles of artist, artwork, selection, static, itinerary or other pages that already carry specific titles, and of fixed-label pages such as Timeline or Glossary.
- Changing when the global search pushes a URL.

## Capabilities

### New Capabilities

- `listing-page-titles`: state-derived document titles for the artist index, artwork search and global search, including length fitting and HTMX swap behaviour.

### Modified Capabilities

None.

## Impact

- Shared title rendering helpers in the Templ utilities.
- The artists, artworks and search handler packages and their page templates. Each of these fragments begins to emit a `<title>`: `ArtworkSearchBlock`, `ArtworkSearchResults` and `SearchResults`.
- No data, configuration or dependency changes.
