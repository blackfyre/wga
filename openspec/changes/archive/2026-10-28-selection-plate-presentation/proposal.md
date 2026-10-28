# Proposal

## Why

A review on 2026-10-27 found three presentation faults in artist selection plates. Plates do not say where the work is held, although the catalogue stores the current holding for most works. Plates in one row have different heights, so their `VIEW WORK →` and itinerary actions sit at different heights. Plates on a selection page and in the artist record's selection previews have different sizes at the same viewport width: the selection page uses a two-column/four-column grid, the record previews use a one-to-four-column grid, and the two grids sit in containers of different widths. At 834px, for example, a selection-page image is 92px wide and a record-preview image is 175px wide.

## What Changes

- Selection plates (on the selection page and in the artist record's selection previews) show the work's stored current holding as one quiet text entry, which wraps when long. The entry is omitted when no holding is stored; no location is inferred or invented.
- The selection page and the record's selection previews share one selection plate grid: two equal columns on narrow viewports, and fixed-width plate columns that fill the available width from the medium breakpoint. Plates therefore have the same size in both places at the same viewport width, while the number of plates per row follows the available width. This replaces the reference "two-column/four-column" wording for the selection page grid.
- Every grid that uses the public artwork plate (selection page, selection previews, Inspiration, and the Dual Mode artist pane) aligns its plates: plates in one row have identical image boxes, and their trailing `VIEW WORK →` and itinerary actions (and Inspiration's study-board action) start at the same height, including when itinerary states differ.

Non-goals:

- No change to the catalogue data, the holding relation, or the artwork record's location presentation.
- No location entry on non-selection plate uses (Inspiration, Dual Mode).
- No change to the artist record's `WORKS IN ARCHIVE` grid, which uses a different card.
- The separately reported "VIEW WORK downloads the page" fault is handled as a bug outside this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artist-selections`: selection plates show the stored holding; the selection page and record previews use one size-consistent selection plate grid; plates in a row align their actions.

## Impact

- Templ: the public artwork plate component and its grids, the selection page, the artist record's selection preview block, and the Inspiration grid.
- Handlers: the artist selection read paths resolve the holding name for the works they show, in one bounded lookup per grid.
- Tests: templ rendering tests for the location line, handler tests for the holding projection, and Playwright geometry checks for plate size and action alignment.
- Design parity: the fixed-width grid deviates from the reference two-column/four-column selection grid in the Claude Design project; the reviewer should confirm the deviation.
