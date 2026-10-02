## Why

The design parity audit of 1 October 2026 (items HO-1 to HO-6) found that the home page has drifted from the Claude Design home screen. The home page offers no route to the Timeline, the hero's second action no longer opens Dual Mode, and the feature-card copy, the work-of-the-day caption, and the collection count labels differ from the design. The design is the UI source of truth (`docs/design-sync.md`), except where the repository has corrected facts that the design still states wrongly.

## What Changes

- Reduce the hero to the design's two actions: BROWSE ARTISTS → (primary, `/artists`) and COMPARE TWO WORKS → (secondary, `/dual-mode`). BROWSE ARTWORKS → and FIND INSPIRATION → leave the hero; artworks stay reachable through ALL WORKS → and the primary navigation, and Inspiration through the primary navigation.
- Replace the three feature cards with the design's TWO WINDOWS (OPEN DUAL MODE → to `/dual-mode`), TIMELINE (OPEN THE TIMELINE → to `/timeline`) and POSTCARD SERVICE (SEND A POSTCARD → to `/postcard`, and MEET THE CONTRIBUTORS → to `/contributors`) cards, with the design's copy and type sizes. The COMMUNITY card is removed; its contributors link moves into the POSTCARD SERVICE card.
- Give the work of the day the design's caption: the artist in capitals, the date, and a trailing arrow.
- Rename the count labels to REPRODUCTIONS, ARTISTS, SCHOOLS and CENTURY, keeping the live counts and the corrected period, set in the design's casing as "3rd–early 20th".
- Keep the repository's factual corrections: the kicker says SINCE 1996, and the period runs to the early 20th century.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `collection-discovery`: the home page's discovery routes are the design's hero actions and feature cards, including Dual Mode, the Timeline, postcards and contributors; artwork browsing and Inspiration are reached through ALL WORKS → and the primary navigation rather than hero actions.

## Impact

- `internal/assets/templ/pages/home.templ`, its Go template test, and `playwright-tests/home.spec.ts`. No handler, data, route, or configuration changes.

## Non-goals

- The recent-additions header rule, grid spacing, and the hero plate are unchanged; the audit did not list them.
- The design's stale facts (SINCE 1992, 3rd–19th, and "Period music optional") are not adopted.
