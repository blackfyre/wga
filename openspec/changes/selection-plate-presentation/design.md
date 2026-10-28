# Design

## Context

Measured on 2026-10-28 with real catalogue data (Altdorfer, selection "Altarpieces"), plate image widths were:

| Viewport | Record preview | Selection page |
| -------- | -------------- | -------------- |
| 390px    | 330px (1 col)  | 147px (2 col)  |
| 834px    | 175px (2 col)  | 92px (4 col)   |
| 1440px   | 161px (4 col)  | 177px (4 col)  |

Within one row, `VIEW WORK →` tops differed by up to 70px because the plate stacks its actions directly under a title of variable length.

The two grids sit in page columns of different widths from the medium breakpoint (the record column is beside a 300px rail, the selection column beside a 220px rail), so any grid whose column count is fixed per breakpoint produces different plate sizes in the two places. Below the medium breakpoint both columns span the page width.

## Goals / Non-Goals

**Goals:** equal plate size in both selection places at one viewport width; aligned actions in every plate grid; the stored holding on selection plates.

**Non-Goals:** changing the Inspiration or Dual Mode grid column counts; changing the artwork record.

## Decisions

### Fixed-width plate columns from the medium breakpoint

One shared selection plate grid: two fluid columns below `md`, and `repeat(auto-fill, 11.25rem)` columns with the existing 24px gap from `md`. 11.25rem lets four plates fit both the selection column (860px) and the record column (about 796px) at 1440px, and two plates fit both columns at 834px.

Alternative: keep two/four columns and apply them to the record previews. Rejected because the column widths differ, so plates would still differ (at 834px, 85px against 112px cards).

Trade-off: the grid can leave unused space at the right edge of the column, and this deviates from the reference two/four-column selection grid.

### Card-level alignment

Each plate spans three rows of its grid (record text, `VIEW WORK →`, itinerary action) and adopts them as a CSS subgrid, so all plates in a grid row share the three row heights. A wrapper between the grid and the plate (the selection list item, the Inspiration item that adds a study-board action as a fourth row) passes the rows through as a subgrid too. The grids drop their row gap, because a parent row gap would also open between a plate's own rows; each item's bottom padding provides the space between plate rows instead. Outside a grid, the subgrid falls back to an ordinary stack. This applies to every grid that uses the plate, which is intended.

Alternative: a full-height flex column with the action group pushed to the bottom. Rejected because the actions then align only when their own heights match: an `IN YOUR ITINERARY ✓` label on one line beside a wrapped `ADD TO AN ITINERARY +` label offsets both actions.

### Holding name from the current-location relation, resolved per grid

The location line uses the artwork's current-location relation, the same field as the artwork-search venue facet and the artwork record's `CURRENT LOCATION`. The comment-derived location is not used, so no location is parsed from free text. The selection read paths resolve all holding names for one grid in one lookup rather than one lookup per work. A failed lookup omits the lines rather than failing the page, matching the artwork record's behaviour for an unresolvable holding.

## Risks / Trade-offs

- Long holding names wrap and increase plate height; alignment keeps actions level, at the cost of taller rows.
- Pinned selection-page tests that count columns or assert the two/four-column classes need updating.
