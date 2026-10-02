## Why

PR #244 put the remembered artwork search choices behind the optional `preferences` cookie-consent category. The site's other remembered preferences still read and write browser storage without consent: the colour scheme, the palette, bionic reading, and the Study Board. These are optional conveniences, not storage the site needs to work, so they belong in the same optional category. The consent notice also describes the category as covering only the search toolbar, which is no longer accurate.

## What Changes

- Extract the consent gate from `resources/js/search-prefs.ts` into a shared module. The module reads `cc_cookie` live, offers gated storage reads and writes, and tells each registered store when consent is first granted or is withdrawn.
- Gate `wga-theme`, `wga-palette`, `wga-bionic` and `wga-study-board` on that consent, alongside the existing `wga-aw-prefs` and `wga_aw_prefs`.
  - **Without consent:** the controls work for the current page, but nothing is read or written. Any stored copy is deleted.
  - **On a first grant:** each store saves the state the page currently shows.
  - **On withdrawal:** every copy is deleted, and the open page keeps its state.
- The inline pre-paint head resolver reads the stored palette and scheme only while `cc_cookie` accepts `preferences`. Otherwise it renders the default palette and follows `prefers-color-scheme`.
- Expire the legacy `wga_theme`, `wga_palette` and `wga_bionic` cookies, which earlier releases set and nothing now reads.
- A `?board=` URL no longer replaces a remembered Study Board. Editing the board still saves it, while consent exists. This matches the rule for explicit search URLs.
- Update the consent notice, the preferences dialog, the appearance panel's storage note, and the Study Board page copy, so they describe what is now remembered and when.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: the `preferences` consent category covers every remembered preference. Palette and scheme are remembered only with that consent.
- `bionic-reading`: the bionic-reading choice is remembered only with preference consent.
- `study-board`: the board is remembered in the browser only with preference consent, and a URL board does not replace the remembered board.

## Impact

- `resources/js/`: a new `preference-consent.ts`, plus changes to `search-prefs.ts`, `appearance.ts`, `bionic.ts`, `study-board.ts` and `cookieconsent.ts`.
- `internal/assets/templ/dto/preferences.go` (the head resolver), `components/footer.templ` and `pages/study_board.templ`.
- Bun, Go and Playwright tests for these modules, and the existing specs that relied on remembered appearance, bionic or board state.
- No route, handler, data or configuration changes. The server reads no appearance, bionic or board cookie, before or after this change.
