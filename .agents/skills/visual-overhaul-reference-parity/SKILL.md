# Visual Overhaul Reference Parity

Use this skill when aligning WGA's shared visual shell with the authoritative
prototype at `wga-visual-overhaul/project/WGA Prototype.dc.html`.

## Source of truth and scope

- Treat the prototype's explicit dimensions, breakpoints, and component
  structure as authoritative.
- Preserve deliberate current-project behaviour that the task explicitly keeps
  (for example, the preferences panel's sticky header).
- Compare the rendered interaction as a whole: breakpoint -> Templ markup ->
  CSS -> HTMX/JavaScript lifecycle -> browser geometry. Do not copy a visual
  value without checking its surrounding interaction contract.
- Change authoritative `.templ`, `resources/css/style.pcss`, and
  `resources/js/` sources only; regenerate outputs instead of editing them.

## Shared container and top navigation

- Shared content geometry is a `1240px` outer cap, including gutters:
  - below `720px`: `20px`;
  - `720px–1079px`: `28px`;
  - `1080px+`: `40px`.
- The navigation changes from mobile disclosure to desktop at `720px`, not
  Tailwind's default `md` breakpoint (`768px`). Use arbitrary variants such as
  `min-[720px]:flex` where the component must follow the prototype.
- Desktop search is `190px` from `720px` through `1079px`, then `340px` from
  `1080px`. Do not use Tailwind's default `lg` breakpoint (`1024px`) for this
  transition.
- Keep the mobile menu functional below the actual `720px` threshold and retain
  real anchor/form fallback behaviour.

## Preferences sheet

- The footer trigger opens a full-height panel:
  - below `720px`: full viewport width;
  - `720px+`: `400px`, right-aligned.
- Use a native `<dialog>` in the top layer with an opaque backdrop. The panel
  itself is the scrollport.
- Preserve the sticky preferences header, native dialog focus restoration,
  Escape dismissal, and existing palette/scheme/bionic data attributes.
- Test both 390px and 720px geometry, right alignment, full height, and header
  position after scrolling the panel.

## Itinerary viewer transitions

- Public itinerary stops remain server-rendered at
  `/itineraries/{token}?stop=N`; `href` is the no-JavaScript fallback.
- Enhanced stop links must swap only `#itinerary-viewer` with:

  ```html
  hx-target="#itinerary-viewer"
  hx-select="#itinerary-viewer"
  hx-swap="outerHTML"
  hx-push-url="true"
  ```

- Do not replace `#mc-area` for a stop. Replacing the shell repaints the page
  beneath the fixed dark viewer.
- `.wga-viewer` must stay opaque: do not attach an entry fade that replays for
  every stop. Since the shell enables HTMX View Transitions globally, bind a
  viewer-local `htmx:beforeTransition` listener in `resources/js/itinerary.ts`
  that calls `preventDefault()`.
- Keep keyboard navigation as a real click on the same stop link and prefetch
  no more than the two neighbours.

## Verification

Start with the affected source and test scope, then build generated assets.

```sh
templ generate
bun run build
go test ./internal/assets/templ/components -run '^TestTopNav|^TestFooter'
go test ./internal/assets/templ/pages -run '^TestItineraryView'
go test ./internal/handlers/itineraries
bun test resources/js/itinerary.test.ts
bunx biome check resources/js/itinerary.ts resources/js/itinerary.test.ts
```

Use targeted Chromium tests for each changed surface. For itinerary slides,
assert the request target is `itinerary-viewer`, `#mc-area` remains mounted, the
viewer has no opacity animation, and the URL still changes. Report unrelated
stateful/parallel browser-test failures separately from the scenario under
test.
