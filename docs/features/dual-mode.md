# WGA Dual Mode Feature Spec

> Target: Provide a desktop-only split-pane browsing experience for comparing and studying artist and artwork content side by side. Current route: `/dual-mode`.

---

## 1. Summary

Dual Mode lets visitors open two related pieces of content in parallel within a single desktop browsing view. It is intended for study, comparison, and cross-reference workflows where a user benefits from keeping two pages visible at the same time.

This feature applies to:

- artist pages
- artwork pages
- artist biography content shown within artist pages

Dual Mode is not intended for mobile use. Small screens do not provide enough space for two usable panes.

---

## 2. Audience and Use Cases

**Audience**

- visitors exploring related artists
- visitors comparing artworks
- visitors reading an artist biography while viewing artworks or another artist

**Primary use cases**

- compare two artists side by side
- compare two artworks side by side
- read an artist biography while keeping an artwork visible
- keep one pane on an artist page while the other pane follows related artwork navigation

---

## 3. Supported Content

### Supported pane content

- artist detail pages
- artwork detail pages
- biography content that is part of an artist page

### Not in scope

- a separate standalone bio-page route
- mobile split-pane layouts
- admin or editorial workflows
- list pages, search result pages, or unrelated static pages unless added later by a separate feature decision

---

## 4. Experience Rules

### Layout

- Dual Mode presents two panes in a single desktop viewport.
- Each pane can render supported content independently of the other pane.
- The page should support left/right pane combinations of any supported content type.

### Entry and navigation

- Users can open Dual Mode from the dedicated `/dual-mode` route.
- The view should preserve which content is loaded in the left and right panes through the URL so the state can be reloaded or shared.
- If one pane is missing content, Dual Mode should still render a usable default state rather than failing the entire page.

### Desktop-only behavior

- Dual Mode is available only on desktop-sized screens.
- On screens below the supported breakpoint, users should see a clear unsupported-state message instead of an active split-pane layout.
- The current minimum supported width should align with the existing desktop breakpoint in the UI unless changed by a later product decision.

### Content behavior

- Artist biography content counts as supported Dual Mode content when shown inside the artist page.
- Dual Mode should prioritize readable browsing and comparison, not editing or complex multi-step workflows.
- Pane content should remain visually independent so users can scroll and inspect each side separately.

---

## 5. Acceptance Criteria

- [ ] Visiting `/dual-mode` on a desktop-sized screen renders a two-pane browsing layout.
- [ ] A user can load an artist page into either pane.
- [ ] A user can load an artwork page into either pane.
- [ ] A user can view artist biography content as part of an artist page in either pane.
- [ ] Left/right combinations of supported content types work without forcing both panes to be the same type.
- [ ] Reloading or sharing the Dual Mode URL preserves the pane state.
- [ ] If one pane has no selected content, the page still renders with a valid default state for that pane.
- [ ] On screens below the supported desktop breakpoint, the split-pane interface is not shown.
- [ ] On unsupported small screens, the user sees a clear message explaining that Dual Mode is desktop-only.

---

## 6. Non-Goals

- building a mobile version of Dual Mode
- defining a separate biography route outside the artist page
- introducing admin-only Dual Mode behavior
- expanding Dual Mode to every content type in the site without a separate product decision

---

## 7. Notes for Future Follow-Up

- If the product later adds dedicated entry points into Dual Mode from artist or artwork pages, that should be captured as a follow-up enhancement.
- If tablet support is ever desired, it should be treated as a new scope decision rather than implied by this spec.
- If additional content types become eligible for split-pane browsing, the supported-content section and acceptance criteria should be expanded together.
