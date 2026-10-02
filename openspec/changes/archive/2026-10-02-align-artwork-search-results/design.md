## Context

Artwork search renders its pagination through `internal/utils/pagination.go`, a generic HTML-string renderer. It is used only by `buildArtworkSearchResults`. The artist listing already renders the design's `PAGE n OF m · ← PREV · NEXT →` row in `artists.templ`. The design shows the same row on both screens.

The SCHOOL and FORM facets already order their options by count descending, with label ascending for ties, in `buildCountedMultiFacet`. This is the order the `catalogue-exploration` specification requires. Audit item AS-3 therefore concerns presentation, not ordering.

## Goals / Non-Goals

**Goals:**

- Match the design's pagination row, facet order, facet option rows and empty-state copy.
- Keep every interaction working without JavaScript.

**Non-Goals:**

- Change the artist listing. It already matches the design.
- Restore TONE. Tone-keyword filtering stays deferred, as the specification states.
- Change the result count, sort, view or Dual Mode contracts.

## Decisions

### Pagination is a server-rendered Templ row with real links

The results view gains `Page`, `PageCount`, and previous and next URLs instead of a pre-rendered HTML string. `ArtworkSearchResults` renders the row only when there is more than one page, as the artist listing does.

An available direction is `<a href="/artworks?…&page=n">`. The same element has `hx-get="/artworks/results?…&page=n"`, `hx-target="#artwork-search-results"`, `hx-select="#artwork-search-results"` and `hx-swap="outerHTML"`. These are the request and target the old renderer used, so the HTMX contract and the pushed URL are unchanged. An unavailable direction is a `<span aria-disabled="true">` in `--faint-2`. It is never an `<a>` without `href`.

The numbered page links of the old renderer are dropped, because the design has none. `internal/utils/pagination.go` has no other caller, so it is removed.

Alternatives considered:

- Keep the generic renderer and restyle it. Rejected: it builds HTML by string concatenation and unescapes its own output, and nothing else needs it.
- Share one component with the artist listing. Rejected for now: the two screens use different targets and URL builders, and the artist row is already correct. A shared component would only move markup that already matches.

### Facet options are text rows that wrap a visually hidden native checkbox

Each SCHOOL and FORM option is a `<label>` row that contains an `sr-only` native `<input type="checkbox">`. The visible text is the label, plus a `×` marker when the option is selected, and a count. A native checkbox gives the row the checkbox role and checked state that the design describes with `role="checkbox"` and `aria-checked`. It also keeps keyboard toggling with Space and the form's no-JavaScript submit. Focus is shown on the row through `:has(:focus-visible)`. A zero-result option stays a disabled row, as the specification requires.

The server already sorts by count, so the order is not changed.

### TECHNIQUE and PERIOD stay, between TYPE and YEAR RANGE

The design's order is TITLE OR ARTIST, COLLECTION, SCHOOL, FORM, TYPE, TONE, YEAR RANGE. TECHNIQUE and PERIOD are working filters required by the `catalogue-exploration` specification, so they are kept. They take TONE's retired position, between TYPE and YEAR RANGE. The design's own facets keep their relative order, and PERIOD sits next to YEAR RANGE, the other temporal filter.

### Empty-state copy

The filtered empty state uses the design copy. The browse prompt for an unfiltered empty catalogue is unchanged, because the design has no equivalent state. The shared `EmptyState` component already has the design's spacing, type sizes and right-aligned `RESET FILTERS →` action.

## Risks / Trade-offs

- Visitors can no longer jump directly to a numbered page. → The design deliberately offers only previous and next. A `page` URL parameter still addresses any page directly.
- A visually hidden checkbox depends on a clear row state. → The selected row is semibold, shows the `×` marker and exposes its checked state to assistive technology.
