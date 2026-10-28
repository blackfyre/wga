# Proposal

## Why

Visitors choose between the grid and list presentation on several listing screens, but only artwork search remembers that choice, and only on a bare `/artworks` visit. The artist index always opens as a grid and Dual Mode index panes always open as a list, so a visitor who prefers lists has to switch again on every screen and every visit.

## What Changes

- One remembered result view (`grid` or `list`) is shared by every screen that offers a grid/list toggle. It is stored in the existing `view` field of the `wga_aw_prefs` preference (cookie and `localStorage["wga-aw-prefs"]` mirror); no new cookie or storage key is added.
- Screens in scope, which are all the grid/list toggles in the application:
  - artwork search (`/artworks` and its `/artworks/results` fragments);
  - the artist index (`/artists`);
  - both Dual Mode artist-index panes (`l_view` and `r_view`).
- Out of scope: the Study Board MATRIX/BOARD switch (not a grid/list choice), the tour `list` page type (a tour page kind, not a toggle), and artist record and selection pages.
- When a request for an in-scope screen carries no explicit view parameter for that listing, the server renders the remembered view when preference consent exists and a valid value is stored; otherwise it renders the grid. An explicit view parameter always wins.
- For artwork search the remembered view now also applies to non-bare `/artworks?...` URLs and to `/artworks/results` fragments that carry no `view` parameter. Remembered sort, direction, and result-actions semantics are unchanged (sort and direction still apply only to a bare `/artworks` visit).
- Activating any view toggle on any in-scope screen records the addressed view in `wga_aw_prefs` before the link is followed, preserving the other stored fields. Consent rules are unchanged: without `preferences` consent nothing is read or written.
- Links rendered while a visitor sees the grid although their remembered view is list carry an explicit grid parameter, so paging, filtering, and sorting keep the view the visitor is looking at.
- **BREAKING (behaviour):** Dual Mode index panes default to grid instead of list. Canonical Dual Mode URLs therefore omit `grid` (not `list`) for a pane, and a bookmarked Dual Mode URL without `l_view`/`r_view` now opens that pane as a grid for visitors without a remembered list view.
- Responses whose view depends on the preference cookie vary on `Cookie` and are never shared from a cache when the cookie is present.
- User-facing copy describing `wga_aw_prefs` as artwork-search-only (cookie preferences dialog and the privacy policy's cookie section) is updated to say the remembered view applies to every listing.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: the artwork-search remembered-choices requirement widens the view rule to every request without an explicit view parameter, and a new requirement makes the artist index and Dual Mode index panes inherit and record the same remembered view, with grid as the default everywhere.

## Impact

- Handlers: `internal/handlers/artworks`, `internal/handlers/artists`, `internal/handlers/dual`, plus a small shared reader of the preference cookie and consent record.
- Templates: `pages/artworks.templ`, `pages/artists.templ`, `pages/dual.templ` (toggle markers and view hidden inputs).
- Browser: `resources/js/search-prefs.ts` records views from the new generic toggle marker; `resources/js/cookieconsent.ts` copy.
- Data: one idempotent migration that rewords the privacy policy's preference-storage sentence when the shipped wording is present.
- Caching: `/artists` and `/dual-mode` responses gain `Vary: Cookie` and become `private, no-store` when the preference cookie is present; the Cloudflare rule already bypasses the edge for requests carrying `wga_aw_prefs`.
