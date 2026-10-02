## Why

Between 720 and 1079px the desktop header search form is fixed at 190px and none of its three children keeps its natural width. The `CTRL K` keyboard cue is squeezed to 49px, wraps onto two lines and grows to 46px tall, pushing the logo row; the search input shrinks to about 56px, cutting off its placeholder; and at 834px INSPIRATION wraps onto a second navigation line. The design places search and the hotkey on the logo row and the navigation on a single second row. The design audit also found that the wordmark and strapline drift from the design's type, and that MORE shows the browser's native `▶` disclosure marker where the design reads `MORE ▾`.

## What Changes

- The SEARCH and `CTRL K` controls keep their natural width on one line; the search field takes the remaining space up to the existing 340px form maximum, at least 140px wide from 720px, and shrinks rather than overflowing when a classic scrollbar narrows the page.
- The logo row uses the design's 16px gap, and the search controls tighten their gap below 1080px so the field stays readable at 720px.
- The navigation row tightens its item gap below 1080px so all destinations and MORE fit on one line at 834px, even with a classic scrollbar; at 720px it wraps to a second line without dropping items.
- The desktop wordmark and strapline follow the design: `--t-14`, weight 600, 3px tracking and 1.15 line height for the wordmark; `--t-10`, 1px tracking, the faint colour role and a 4px top margin for the strapline, which reads `EUROPEAN ART, 3rd CENTURY – EARLY 20th` on desktop and mobile.
- MORE renders `MORE ▾` without the native disclosure marker.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: the responsive public navigation requirement gains the desktop header layout contract.

## Impact

- `internal/assets/templ/components/nav.templ` only, plus its Go render test and the Playwright header specs. No route, data or behaviour other than presentation changes.
- Non-goals: the mobile header layout and type (only the shared strapline text changes there), the MORE menu's contents, and the keyboard Go to behaviour.
