## Context

Source: Claude Design project "WGA", `CHANGELOG.md` entry of 2 October 2026, "Keyboard bar is permanent; floating controls clear every bottom bar". The bar used to appear after the first key press; it now shows at all times on desktop and stays hidden on mobile. FEEDBACK and the toast stack sit a fixed gap (24px desktop, 16px mobile) above everything docked at the bottom, computed from the total stack height. The cookie notice's own placement, the bar's contents, and the tray heights are unchanged; pages reserve the bar's 30px.

In the repository, `.wga-kbd-bar` is `display: none` unless `html[data-kbd-on]` (set by `keyboard.ts` on first key use) and `(hover: hover) and (pointer: fine)` both hold. The bar is fixed at `bottom: 0` but is not a measured bottom-stack item, so the trays start at the viewport edge and the bar covers their lowest 30px once it appears. FEEDBACK (inline style) and `.wga-toast-stack` use `bottom: calc(var(--wga-bottom-stack-height) + 1rem)`.

## Decisions

### Desktop is a hovering, fine pointer

Keep the existing `(hover: hover) and (pointer: fine)` media query as the only gate on the bar. It is the repository's equivalent of "mobile has no keyboard": it hides the bar on phones and touch tablets at any width and shows it on a narrow desktop window. The bar is shown by CSS alone, so it is present from the first paint, with no flash or mid-session height change. Without JavaScript the bar still shows, which matches the design's "render unconditionally"; its shortcuts need the keyboard script, as they always did once the bar was revealed.

Alternative considered: gate on `html[data-keyboard-navigation-ready]`, set when the keyboard script initialises. Rejected because the bar would appear one script load after first paint on every full page load, changing the stack height after layout.

### The bar is the lowest measured stack item

The bar element gets `data-wga-bottom-stack-item="keyboard"` and `data-wga-bottom-stack-order="0"`, below the itinerary tray (10), the Study Board tray (20), and the cookie notice (100). `bottom-stack.ts` already skips items with no client rects, so on touch devices the hidden bar contributes nothing, and on desktop the trays rise by its 30px and `--wga-bottom-stack-height` includes it. No script change is needed for measurement. The stylesheet fallback used before the first measurement adds 30px under the same media query, so content reservation is correct before the script runs.

### `data-kbd-on` is removed

The flag's only reader was the bar's display rule; carets use `data-kbd-caret` and the dialogs their own state. With the bar unconditional the flag has no consumer, so `markUsed()` and its calls are removed rather than left as dead state.

### One floating gap token, 16px below MD and 24px from MD

A `--wga-floating-gap` custom property is 16px below 720px and 24px from 720px (the repository's `--breakpoint-md`, 45rem). FEEDBACK and `.wga-toast-stack` both use `bottom: calc(var(--wga-bottom-stack-height, 0px) + var(--wga-floating-gap))`, so they move together. MD and LG count as desktop for the gap because 720px is already where FEEDBACK's right inset widens from 16px to 32px (`md:right-8`) and where the itinerary tray drops to its single-row height; tying the bottom gap to the same tier keeps the corner control's insets consistent. The SM tier (<720px) is the phone layout the changelog calls mobile. The gap is keyed on width, not pointer, because it describes the corner's proportion, not keyboard availability.

### The reservation moves from `<main>` to `<body>`

The bottom-stack reservation was `padding-bottom` on `#mc-area`, but the shared footer follows `<main>`, so at the end of a page the footer's final row sat under any docked tray; only the footer's own 60px bottom padding hid this for the bar alone. The design states that pages reserve the whole stack, so `wga-bottom-stack-content` moves to the shared layout's `<body>`, putting the reserved space after the footer. `<body>` is not replaced by HTMX swaps, so the reservation needs no per-fragment markup; the artist record fragment drops its copy of the class.

Alternative considered: `body:has(> .wga-bottom-stack-content)` in CSS, leaving the markup alone. Rejected because browsers without `:has()` would lose the reservation entirely.

## Risks / Trade-offs

- Desktop pages lose 30px of viewport permanently. This is the intended design trade for a stable stack.
- The cookie notice now rises 30px on desktop, because it sits 16px above the stack beneath it, which now includes the bar. Its own placement rule is unchanged.
