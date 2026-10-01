## Why

In-page anchor links (the artist "ON THIS PAGE" rail, the Dual Mode pane contents, static-page tables of contents and the footer back-to-top link) jump abruptly, which loses the visitor's sense of where they moved to. Conversely, an HTMX navigation that replaces the main area keeps the previous page's scroll offset, so a visitor who follows a header link from the bottom of a long page lands mid-way down the new page.

## What Changes

- Scroll smoothly for in-page anchor navigation in the document and in Dual Mode pane scroll regions.
- Honour `prefers-reduced-motion: reduce` fully: no smooth scrolling and an instant return to the top.
- Return the viewport to the top after an HTMX navigation that swaps the main area (`#mc-area`) and pushes or replaces the URL.
- Leave fragment and in-place updates (search results, artwork-search results, Dual Mode panes, itinerary tray, study board shelf, toasts and dialogs) at their current scroll position.
- Keep HTMX history scroll restoration on Back and Forward, applied instantly, and land on the named section for `/page#section` navigations.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: adds smooth in-page scrolling and the post-navigation scroll position contract.

## Impact

- `resources/css/style.pcss` (scroll behaviour), a new `resources/js/navigation-scroll.ts` module initialised from `resources/js/bootstrap.ts`, and a Playwright spec.
- No server, template, data, or API changes.
