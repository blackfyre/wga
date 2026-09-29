# listing-page-titles Specification

## Purpose
Gives the artist index, artwork search and global search document titles that describe the active query, filters and page position. Tabs, history entries and screen-reader page announcements can then tell listing states apart.

## Requirements

### Requirement: Listing titles describe the active state

The document title of the artist index (`/artists`), artwork search (`/artworks`) and global search (`/search`) SHALL describe the listing state that is shown. When no state-contributing value is active and the first page is shown, the title SHALL remain the page's existing fixed title.

The following values SHALL contribute to the title, and no others:

- **Artist index:** the name query, the selected letter, the selected school, the selected period, the birth-year range when narrowed, and the page position.
- **Artwork search:** the exact artist scope (by public filing name), the free-text query, the title, artist-text and technique filters, the school, form, type, period, collection and year selections, and the page position.
- **Global search:** the search term.

Result view, sort order, result counts and Dual Mode hand-off state SHALL NOT appear in the title.

#### Scenario: Unfiltered listing keeps its fixed title

- **WHEN** a visitor opens `/artworks` with no filters on the first page
- **THEN** the document title is `Artworks Search - WGA`

#### Scenario: Artist index letter and school

- **WHEN** a visitor opens the artist index with letter `B` and a selected school, on the first page
- **THEN** the document title contains the letter and the school's display label, and the page name `Artists`

#### Scenario: Global search term

- **WHEN** a visitor opens `/search?q=giotto`
- **THEN** the document title begins with the quoted term `“giotto”` and contains `Search`

#### Scenario: Presentation-only state is excluded

- **WHEN** a visitor switches an artwork search between grid and list view or changes its sort order
- **THEN** the document title is unchanged

### Requirement: Listing title format

A listing title SHALL be composed as follows:

1. The distinguishing part comes first, when present. This is the free-text query or search term in typographic quotes, or the exact artist scope.
2. The page name follows (`Artists`, `Artworks Search` or `Search`).
3. Further filter parts follow, in the order their controls appear on the page.
4. The page position is written `p. N/M` and comes last.

Parts SHALL be separated by `·` and followed by the existing ` - WGA` suffix. The page position SHALL be omitted on the first page. Filter values SHALL use their display labels in their normal letter case, not the uppercase facet summaries. Visitor-supplied text SHALL render as escaped text and SHALL NOT be interpreted as markup.

#### Scenario: Query, facet and page position

- **WHEN** a visitor views page 3 of 41 of `/artworks` with the query `madonna` and the school `Florentine`
- **THEN** the document title is `“madonna” · Artworks Search · Florentine · p. 3/41 - WGA`

#### Scenario: First page omits the page position

- **WHEN** a visitor views page 1 of an artist index filtered by letter `B`
- **THEN** the document title contains no `p.` part

#### Scenario: Markup in a query is escaped

- **WHEN** a visitor opens `/artworks?q=<b>`
- **THEN** the response's `<title>` contains `&lt;b&gt;` and no element is created from the query

### Requirement: Listing titles fit a maximum length

A listing title, including the ` - WGA` suffix, SHALL NOT exceed 80 characters, counted as Unicode code points. When the full title would exceed that length, the system SHALL reduce it by applying a fixed ordered sequence of reductions, stopping at the first result that fits. Each reduction replaces a part with its short form or removes a part. The page name and the ` - WGA` suffix SHALL never be removed. When several filter parts are removed, they SHALL be replaced by a single count part (`+N`) rather than disappearing without trace. The free-text part SHALL be the last to be shortened, and SHALL be shortened by truncation with a trailing `…`. The same state SHALL always produce the same title.

#### Scenario: Long state is reduced within the limit

- **WHEN** an artwork search has a long query, several facet selections and a page position whose full title exceeds 80 characters
- **THEN** the document title is at most 80 characters, still contains the page name and the ` - WGA` suffix, and starts with the query or its truncated form

#### Scenario: Removed filters are counted

- **WHEN** reductions remove two filter parts from an artwork-search title
- **THEN** the title contains a `+2` part in their place

#### Scenario: Short titles are not reduced

- **WHEN** the full title of a listing state is within 80 characters
- **THEN** every part appears in its long form

### Requirement: Enhanced navigation keeps listing titles current

When an enhanced (HTMX) request on a listing page swaps in results for a new state, the document title SHALL be updated to the title of that state. This includes artwork-search result, sort, view and pagination swaps, artist-index filter swaps, and global-search result swaps. Full page loads without JavaScript SHALL render the same title for the same state.

#### Scenario: Artwork filter swap updates the title

- **WHEN** a visitor with JavaScript enabled selects a school facet on `/artworks`
- **THEN** after the results swap, the document title includes the school and matches the title a full load of the pushed URL renders

#### Scenario: Global search typing updates the title

- **WHEN** a visitor types `giotto` into the global search field and the debounced results swap in
- **THEN** the document title begins with `“giotto”`

### Requirement: Link previews use the listing title

The `og:title` and `twitter:title` values of a listing page SHALL equal its document title without the ` - WGA` suffix, as for other pages today.

#### Scenario: Shared filtered search

- **WHEN** a link-preview client fetches `/artworks?q=madonna`
- **THEN** the page's `og:title` and `twitter:title` are `“madonna” · Artworks Search`
