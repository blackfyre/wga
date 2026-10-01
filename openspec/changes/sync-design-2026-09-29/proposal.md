## Why

The Claude Design changelog of 29 September 2026 makes two public-surface changes. Every artwork search result currently carries an `ADD TO ITINERARY` / `ADD TO STUDY BOARD` pair, which crowds the results for visitors who are only browsing, and the sort and view choices are lost on each fresh visit. Separately, the design moves "back to top" into the footer as a plain link; the repository still ships a dead floating back-to-top handler and style that nothing renders.

## What Changes

- Hide the per-result itinerary and Study Board actions on artwork search by default, and add an `ACTIONS +` / `ACTIONS ✓` toggle to the sort/view toolbar.
- Merge the separate `GRID` and `LIST` controls into one control that names the current view (`VIEW: GRID` / `VIEW: LIST`) and switches to the other.
- Remember the sort key, sort direction, view, and actions setting on the visitor's device across page loads. A bare `/artworks` visit uses the remembered choices; any explicit query parameter wins, and resetting filters leaves the remembered choices untouched.
- Add a plain `↑ BACK TO TOP` footer link beside `PREFERENCES` that targets the page header, and remove the unused floating back-to-top script and style.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: opt-in result actions, the merged view control, and remembered artwork search presentation.
- `public-page-experience`: footer return-to-top link.

## Impact

- Artwork search handler, its templates, shared layout and footer templates, one new browser module, and shared CSS.
- One new first-party preference cookie (`wga_aw_prefs`) and `localStorage` key (`wga-aw-prefs`); no data, API, or configuration changes.

## Non-goals

- The itinerary and Study Board trays, the artwork record page's own add controls, and the artist index view switch are unchanged.
- Dual Mode search results keep their existing pane-routing cards.
