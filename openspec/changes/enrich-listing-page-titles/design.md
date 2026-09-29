# Design

## Context

- Handlers set the title with `DecorateContext(ctx, TitleKey, v)`, which also sets `og:title` and `twitter:title`. `GetTitle` renders `v + " - WGA"` into the layout's `<title>`.
- `ArtistsBlock`, `ArtistRecordBlock` and `ArtworkBlock` already emit `<title>` for HTMX requests. `ArtworkSearchBlock`, `ArtworkSearchResults` and `SearchResults` do not.
- htmx 2.0.10 updates `document.title` from a response only when `<title>` is a **root-level** element of a partial response. A `<title>` nested inside the swapped markup is ignored by htmx and left in the body.
- `ArtworkSearchResults` is rendered on its own (for `/artworks/results`), inside `ArtworkSearchBlock`, and inside the full page. A `<title>` placed inside the component would therefore be nested in two of those three cases.
- The artwork facet summaries (`FLORENTINE`, `“madonna”`, `2 SELECTED`) are uppercase display strings for the filter rail. They do not suit titles.
- The global search does not push a URL on debounced typing.

## Goals / Non-Goals

**Goals:**
- A single shared fitting mechanism. Each feature package owns its title parts and their priorities.
- Tuning the reduction order means editing one ordered list per page and its table test.

**Non-Goals:**
- Changing `GetTitle`, the ` - WGA` suffix, or titles on other pages.
- Changing the global search to push a URL.

## Decisions

### Shared mechanism, feature-owned policy

A pure function in `internal/assets/templ/utils` takes an ordered list of parts and an ordered list of reduction steps, and returns the fitted title text without the suffix. The suffix length is included in the budget. Each part has:

- a long form;
- a short form, which may equal the long form;
- a role: lead (free text), page name (fixed) or filter.

Each feature package (`artists`, `artworks`, `search`) builds its parts from its own filter state, using display labels in normal case rather than the uppercase facet summaries. The handler passes the result to `DecorateContext(TitleKey, …)` exactly as today, so the link-preview coupling is unchanged.

**Alternative considered:** two fixed phases (shorten every part, then drop parts), driven by one priority number per part. This is simpler to declare, but it cannot express interleaving such as "drop the page position before shortening the query". An explicit ordered list of steps can express this and is still plain data.

### Initial reduction order

1. Remove the page position.
2. Use short forms for filter parts, starting from the last. A multi-value facet shortens to its first value plus `+N`, for example `Florentine +1`. A range shortens to its bounds.
3. Replace filter parts with a single `+N` count, starting from the last.
4. Truncate the lead parts to their step limits plus `…`. When artwork search has both an artist scope and a query, the scope is truncated first, so the free-text query is the last part to be shortened.
5. As a final guard, remove any remaining filters, then truncate the leads from the last one back until the title fits the budget.

A filter value without a resolvable display label, such as a stale slug or a period ID beyond the bounded period projection, is left out of the title rather than shown raw.

The budget is 80 code points including ` - WGA`. It serves as a guard against runaway titles. It is not a search-engine target, because tabs truncate far earlier and the order of parts matters more there.

### Title emitted at the response root for HTMX swaps

The HTMX response paths render a root-level `<title>` before the swapped markup:

- the artwork-search block;
- the `/artworks/results` fragment;
- the `/search/results` fragment.

The title is emitted by a small wrapper at the handler's render entry, not inside the reusable `ArtworkSearchResults` or `SearchResults` components, so embedding those components never nests a `<title>`. The artist index already emits its title from the block root and only needs the enriched value.

**Alternative considered:** setting the title with an `HX-Trigger` event and a JavaScript listener. This was rejected because htmx already handles a root-level `<title>` natively, and server-rendered markup is the project's preferred mechanism.

### Global search title follows the displayed results

On debounced typing, `/search/results` updates the title to the displayed term, while the URL keeps the originally loaded term. The title describes the content that is shown. Aligning the URL is a separate decision and is outside the scope of this change.

## Risks / Trade-offs

- [The tab title and URL disagree after typing in the global search] → This is accepted and documented above. The previous behaviour already left the URL stale.
- [An exact artist scope with a long filing name crowds out the filters] → Artist scope is a lead part, so it is shortened only after the filters have been reduced. The table tests cover a long filing name.
- [Titles change on every facet toggle, and screen readers may announce them] → Only HTMX swaps that change the listing state change the title. View and sort changes leave it unchanged, as the spec requires.
