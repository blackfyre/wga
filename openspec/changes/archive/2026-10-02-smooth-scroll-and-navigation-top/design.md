## Context

`LayoutBase` sets `hx-target="#mc-area" hx-select="#mc-area" hx-swap="outerHTML"` on `<body>`, so header links and record links that carry only `hx-get` replace the main area. Handlers declare the URL with `HX-Push-Url`. HTMX 2.0.10 runs with `globalViewTransitions`, its default `scrollBehavior: 'instant'`, and no `hx-boost`, so no swap currently changes the scroll position. No rule sets `scroll-behavior: smooth`; the reduced-motion block in `style.pcss` already forces `scroll-behavior: auto !important`.

### Swap-target inventory

| Target                                                                                                             | Initiators                                          | URL update            | Returns to top |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------- | --------------------- | -------------- |
| `#mc-area` (body default, explicit artworks RESET, record links whose `HxTarget` is `#mc-area`)                    | header and footer navigation, record links, RESET   | `HX-Push-Url`         | yes            |
| `#mc-area` without a URL update (form posts that inherit the body target)                                          | form submissions                                    | none                  | no             |
| `#global-search-results`                                                                                           | search-as-you-type                                  | none                  | no             |
| `#artwork-search` / results, `#artists`, `#timeline`, `#glossary`, `#guestbook`, `#tour`, `#tours`, `#inspiration` | filters, letters, view toggles, in-block pagination | usually `HX-Push-Url` | no             |
| `#dual-area`, `#dual-left`, `#dual-right` (`w.SelfSel`, `w.TargetSel`, `htmx.ajax` in `updateDualMode`)            | pane navigation and filters                         | push                  | no             |
| `#itinerary-viewer`                                                                                                | stop navigation                                     | `hx-push-url`         | no             |
| `#itinerary-tray`, `#itinerary-builder`, `#postcard-compose`, study board shelf                                    | in-place editing                                    | none                  | no             |
| `#d` (dialog), `#toast-container`                                                                                  | dialogs, toasts                                     | none                  | no             |

Fragment targets push URLs too, so "the URL changed" alone cannot identify a page navigation; the swap target must also be `#mc-area`.

## Goals / Non-Goals

**Goals:** smooth in-page anchor scrolling in the document and in Dual Mode panes; a top-of-page start after a main-area navigation; full reduced-motion compliance; unchanged history restoration and hash landing.

**Non-Goals:** changing which targets handlers swap, adding `hx-boost`, or animating the post-navigation reset.

## Decisions

### Smooth anchors scoped to the fragment jump

A delegated `click` listener on `document` recognises an unmodified primary click on a link to a fragment of the current document (same origin, path and query). Unless reduced motion is requested, it adds `.wga-smooth-scroll` to `<html>` and lets the browser perform its native fragment navigation. Under `@media (prefers-reduced-motion: no-preference)`, that class sets `scroll-behavior: smooth` on `html` and on the Dual Mode pane scroller `.wga-dual-body`. The property is not inherited, and each pane scrolls independently. The class is removed on the next `scrollend`, observed by a capture listener so pane scrolls count, or after 1.5 s.

- Native semantics are kept: hash update, history entry, `:target`, focus starting point and `scroll-margin`. Keyboard activation dispatches `click`, so it is covered. The footer `#top` link needs no code.
- The existing reduced-motion rule (`scroll-behavior: auto !important`) remains the guard for every element.

Alternative rejected: permanent `scroll-behavior: smooth` on `html`. It animates every scroll that omits `behavior`, including HTMX's Back restoration (`window.scrollTo(0, y)`) and browser automation's scroll-into-view. In the suite, Playwright's no-JavaScript timeline submission never found a stable button and timed out. Undoing those effects needs one workaround per caller.

Trade-off: without JavaScript, anchors jump instantly. This is the progressive-enhancement baseline.

### Return to top through one delegated `htmx:beforeHistoryUpdate` listener

`resources/js/navigation-scroll.ts` listens on `document` for `htmx:beforeHistoryUpdate` and calls `window.scrollTo({ top: 0, behavior: "instant" })` when `detail.target.id === "mc-area"` and `detail.history.type` is `push` or `replace`.

- The event fires only for request-driven history updates. HTMX saves the outgoing page and its scroll offset to the history cache before raising it, so Back still restores the previous position.
- It fires inside `swap()`'s `beforeSwapCallback`, in the same task as the DOM replacement and inside the view-transition update callback, so the new page is never painted at the old offset.
- The listener is on `document`, so it survives every swap and needs no re-initialisation.
- The reset is always instant. A new page should start at its top, as a full document load does. Animating across content that has just been replaced is disorienting, and an instant reset satisfies reduced motion by construction.
- For an HTMX `/page#section` navigation, HTMX scrolls the anchor into view after settle, so the visitor lands on the section.

Alternative rejected: `show:window:top` or `scroll:top` on the body's `hx-swap`. `hx-swap` inherits independently of `hx-target`, so any element that overrides only `hx-target` would inherit the modifier and scroll a fragment update. The modifier is applied in `updateScrollState` after HTMX's anchor scroll, which overrides `#section` landing. It also needs an explicit `hx-swap` audit for every current and future fragment initiator.

Alternative rejected: an `htmx:afterSettle` handler. It works, but it runs after the swap has been painted, and the settled event fires per swapped element. The history-update event is the single signal that combines "target is the main area" with "the URL changes".

## Risks / Trade-offs

- [A scripted scroll during the brief smooth window animates] → the window ends on `scrollend` or after 1.5 s. Main-area navigation clears it and scrolls with `behavior: "instant"`.
- [A main-area swap without a URL update, such as a form post, keeps its offset] → this matches the contract that only navigations reset the position.
- [The footer `↑ BACK TO TOP` link arrives in PR #244] → it is a plain `#top` fragment link, so the delegated listener covers it without further code.
