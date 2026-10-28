# Design

## Context

See proposal.md for motivation and specs/catalogue-exploration/spec.md for the behaviour. Today the `wga_aw_prefs` cookie and the consent check are decoded inside the artwork search handler package. The artist index and Dual Mode handlers parse their view only from the URL, and each canonicalises by omitting its own default (`grid` for `/artists`, `list` for Dual Mode panes). The browser module `resources/js/search-prefs.ts` is loaded on every page and already records sort, direction, and view from `[data-wga-aw-pref]` artwork search links in a capture-phase click listener that runs before HTMX follows the link.

## Goals / Non-Goals

**Goals:**

- One server-side reader of the remembered view, shared by three handler packages without one feature reaching into another's helpers.
- No change to the artwork search sort, direction, and actions semantics, or to consent behaviour.

**Non-Goals:**

- Remembering sort, filters, or letters on the artist index or in Dual Mode.
- Re-rendering a Dual Mode pane that was not swapped when the other pane changes the remembered view; pane links stay built from the state at render time.

## Decisions

### A small shared `visitorprefs` package reads the consent record and the remembered view

The consent and cookie decoding moves from `internal/handlers/artworks/prefs.go` into a new `internal/visitorprefs` package that exposes the consent check, the cookie presence check, and the validated remembered view. Artwork search keeps its own decoding of sort, direction, and actions on top of it. Alternative: let the artist and Dual Mode handlers import the artworks package; rejected because it couples features through another feature's internals.

### A missing view parameter resolves to the visitor's fallback view

Each handler computes a fallback view: the remembered view when consent exists and the value is valid, otherwise `grid`. A request without the listing's view parameter renders the fallback; any explicit value is used as addressed (unknown values render the grid). For artwork search the fallback is threaded to the URL builder through the request context, which avoids changing the many builder signatures used by tests.

### URLs omit grid only when the fallback is grid

A URL without a view parameter means "use my remembered view", so it is ambiguous when a visitor looks at a grid while remembering list. URL builders therefore emit `view=grid` (or `l_view=grid`/`r_view=grid`) exactly when the presented view is grid and the visitor's fallback is list, and always emit list. For visitors without a remembered list view the canonical URLs are unchanged on `/artworks` and `/artists`, and Dual Mode canonical URLs omit grid instead of list. Alternative: always emit the view parameter; rejected because it changes every canonical URL and would make shared URLs override recipients' choices even for the default.

### Toggles carry a generic `data-wga-view-pref` marker

Artist index and Dual Mode toggles carry `data-wga-view-pref="grid|list"`. The existing capture-phase listener in `search-prefs.ts` reads that value, merges it into the stored preferences (keeping sort, direction, and actions), and writes both copies under the existing consent gate; modified clicks are ignored as today. The artwork search view toggle keeps `data-wga-aw-pref`, which already records the view it addresses.

### Caching

`/artists` and `/dual-mode` add `Vary: Cookie` and call the existing private-response helper when the `wga_aw_prefs` cookie is present, mirroring artwork search. The edge rule already bypasses requests carrying that cookie, so no infrastructure change is needed.

## Risks / Trade-offs

- [Bookmarked Dual Mode URLs without `l_view`/`r_view` change from list to grid for visitors without a remembered list view] → Called out as a behaviour change in the proposal; list remains one click away and the choice is now remembered.
- [A pane or page rendered before another tab changed the remembered view keeps links built from the old fallback] → Links remain valid; they only differ in whether grid is explicit. This follows the existing render-time link contract.
- [Privacy policy text lives in the database] → An idempotent migration rewrites only the shipped sentence and leaves edited content untouched.

## Migration Plan

Deploy normally. The privacy policy migration runs on start and is a no-op when the shipped sentence is absent. Rollback restores the previous handler behaviour; the stored preference format is unchanged.
