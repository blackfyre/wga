# Design

## Context

The design entry (Claude Design changelog, 5 October 2026) parses raw selection lines and groups them in the browser, because the prototype has no backend. WGA already serves every published selection as a record: `title`, `context`, `display_title` and `artworks`, each at a stable `/artists/{slug}-{id}/selections/{id}` route. The artist handler (`internal/handlers/artists/artist.go`) loads these records and their works, and `pages.ArtistRecordContent` renders one preview per selection. Dual Mode builds its own previews (`internal/handlers/dual/main.go`, `pages.DualSelectionPreview`) and does not change.

The design's three downstream notes resolve as follows:

- **Load from the database:** the records are already loaded from the database.
- **Placeholder lines:** these were sample data, so nothing needs porting.
- **Tiles open a generic page:** in WGA each tile opens its own selection route.

## Decisions

### Group on the server, in the artists package

The parsing and grouping are a pure function of the ordered display titles. It lives beside the selection loader in `internal/handlers/artists`, with table tests, and returns forms that hold their selections. The template only renders. A client-side port of `parseSelectionLine()` would break the no-JS fallback and duplicate state that the server already has.

The parsing order is fixed for determinism:

1. Strip any trailing count after `·`, a dash or a comma.
2. Try the final bracketed group.
3. Try the first colon.
4. Otherwise use the full title.

Under this order, "Woodcut series: The Revelation of St John (Apocalypse) (1497-98)" parses as form "Woodcut series: The Revelation of St John (Apocalypse)" with qualifier "1497–98". The design does not say which rule applies first. Trying brackets first keeps date ranges as tile labels, which is what the design's examples show.

A selection's count comes from its works, not from its title. WGA titles carry no counts, so the count-stripping rule exists only for parity with the design.

### Cover is the selection's first work

The design keeps covers in an optional map. WGA has no such field, but every selection has ordered works. The first work's thumbnail, through the existing `buildRecordWorkImages` path, is the cover. If it has no image, the tile falls back to the form name. One cover per selection replaces the current four-plate preview, so the record does less image work than before.

### Folding is progressive enhancement on plain links

Every form row is an ordinary fragment link to its block, and every selection entry is an ordinary link to its tile. The server renders foldable lists open, with no toggle. A small `resources/js` module then:

- adds the `+`/`−` button with `aria-expanded`;
- collapses every form except the current one;
- tracks the current form with IntersectionObserver on the form blocks only, not the tiles.

The JavaScript must stay idempotent across HTMX swaps of `#mc-area`, as in the existing modules.

Alternative considered: native `<details>`/`<summary>`. It would fold without JavaScript, but the summary would have to hold the form's fragment link, and a link inside `<summary>` is an interactive-content conflict. The design also specifies `aria-expanded`, which `<details>` does not expose in that form.

### Dual Mode is pinned by specification

`catalogue-exploration` currently says the pane renders the same previews "as the artist record". That clause would silently change meaning, so the delta states the pane's presentation explicitly. The pane code is separate and stays untouched.

### Large holdings start at ten selections

The design applies the index only where an `artistIndex` entry exists and leaves other artists "exactly as before". WGA has no such flag, so a threshold on the number of published selections stands in for it. The threshold is ten, accepted on 5 October 2026. It covers the 40 artists whose previews are long enough to need the index. The 191 artists with two to nine selections, mostly two to four, keep their previews and commentary, where folding would gain nothing. The threshold is a named constant beside the grouping function, so tuning it later only means changing that constant and its tests.

## Risks / Trade-offs

- **How much the title rule folds varies by artist.** Among large holdings, Dürer's 36 selections become 20 rows, Raffaello's 24 become 14, and Memling's 15 become 7. El Greco, Fra Angelico and Piero della Francesca fold nothing. Every large holding still gains the compact tile grid. The unused `context` hierarchy, which Giotto and Michelangelo carry, is a candidate design follow-up.
- **Selection commentary leaves large-holding records.** It remains on each selection page, as does the `Commentary is unavailable` notice. Records below the threshold are unchanged.
- **The synthetic fixture has no large holding.** Its curated artist has three selections, so Playwright can only confirm that the preview presentation is unchanged. Exercising the index in a browser needs a fixture artist with at least ten selections, including a shared form, refreshed from `../wga-src`. Until then, Go and templ tests and a `bun test` unit test cover the index, and no browser claim is made for it.
