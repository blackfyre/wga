# Proposal

## Why

The Claude Design changelog entry of 5 October 2026, "Artist page: large holdings get a selection index, grouped in the browser", replaces the artist record's stacked selection previews with an index grouped by form. Large holdings make the current layout unworkable: Michelangelo has 56 published selections and Dürer 36, so the record stacks a commentary block and four plates per selection, and ON THIS PAGE grows far past the viewport.

## What Changes

- **Grouping:** each published selection's title is split into a form, an optional qualifier and the selection's work count. The qualifier sits in brackets or after a colon, and a hyphenated year range is shown with an en dash. Selections are grouped by form, case-insensitively, in the order the forms first appear. A title that does not parse becomes a form of its own under its full title. The grouping is computed by the server from the existing selection records. The producer and the schema do not change.
- **ON THIS PAGE rail:** the rail lists Biography, one row per form (for example `DRAWINGS · 7 SEL · 84`) and Cite This Record.
  - A form with one selection does not fold. Its row links straight to that form.
  - A form with several selections folds behind a `+`/`−` control that reports its state through `aria-expanded`. An open form lists its selections in two columns, and each one links to its tile.
  - Only the form being read is open by default. Scroll tracking works per form and ignores the tile anchors.
- **Main column:** the selection previews are replaced by one block per form. Each block has a `N SELECTIONS · N WORKS` header and a grid of square tiles. Each tile shows the selection's range and count, and links to that selection's existing route. When no cover image resolves, the tile shows the form name instead.
- **BREAKING (presentation), large holdings only:** these records no longer show selection commentary, the four preview plates or `OPEN SELECTION`. The selection page keeps all three.
- **Applies to large holdings: artists with ten or more published selections.** In the current data this is 40 artists. The design states no threshold; ten was chosen on 5 October 2026 from the per-artist distribution. Artists with two to nine selections (191 artists) keep today's previews and ON THIS PAGE, which matches the design's "render exactly as before" clause. Artists with fewer than two selections keep the ordinary works presentation.
- **Dual Mode is unchanged.** The curated artist pane keeps its current previews and contents list.
- **Design sync record:** `docs/design-sync.md` records the entry as implemented.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artist-selections`: the curated-record requirement adds the large-holding threshold and the form-grouped index above it. The ON THIS PAGE requirement adds one row per form for large holdings, with folding and scroll tracking. Below the threshold, both requirements keep their current behaviour.
- `catalogue-exploration`: the Dual Mode curated-pane requirement no longer refers to the artist record's presentation. It states the pane's own previews, which do not change.

## Impact

- `internal/handlers/artists/artist.go`: selection loading and the form grouping, with tests in `artist_test.go` and `artist_record_integration_test.go`.
- `internal/assets/templ/pages/artist.templ` and `artist_test.go`: the rail, the form blocks and the tiles.
- `resources/js/`: a small progressive enhancement for folding and per-form scroll tracking, registered from the main entry.
- `resources/css/style.pcss`: tile and rail rules, if utilities alone are not enough.
- `playwright-tests/artist-record.spec.ts`: the curated record journey.
- **How well the title rule folds varies by artist.** Dürer's 36 selections become 20 rail rows, with 21 selections inside folding forms. Raffaello goes from 24 to 14 rows, Rembrandt from 30 to 20, and Memling from 15 to 7. El Greco, Fra Angelico and Piero della Francesca fold nothing. Every large holding still gains the compact tile grid. The producer's `context` hierarchy is ignored, as in the design.
- **Non-goals:**
  - no producer, schema or migration change;
  - no per-selection cover-image source beyond the selection's own works;
  - no change to the selection page or to Dual Mode.
