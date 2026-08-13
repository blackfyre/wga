## Context

The reference keyboard layer combines discoverable help, section jumps, caret navigation, and a command palette. WGA now has a partial implementation, but it uses a separate browser route map, traverses only artwork results as a flat wrapping list, and exposes record-only palette suggestions. The reference feature descriptions are the behavioural source for the completed layer.

## Goals / Non-Goals

**Goals:**

- Provide accessible, scoped shortcuts without intercepting form editing.
- Make the keyboard layer consistent across supported public screens and responsive tiers.
- Add a rate-limited, server-rendered record suggestion endpoint for the palette.
- Keep section destinations, the help surface, palette rows, and key bindings coherent.

**Non-Goals:**

- Replace browser navigation or add client-side routing.
- Add shortcuts for public screens that do not exist in this application, including Timeline explorer and visitor itineraries.

## Decisions

- Render the keyboard layer once from the public layout. A server-rendered `KeyboardScreens` registry SHALL be the source of each supported section's key, two-digit number, label, and URL; it SHALL generate the palette's section rows, help content, and module data. This prevents browser routes from drifting from public navigation.
- Pages opt into caret traversal using one marked list, `data-kbd-cols`, and row-level caret markup. Up/down and J/K move by the declared column count; left/right move by one record. Movement clamps at the first and last available row instead of wrapping.
- Keep one authoritative keyboard module registered from the existing browser bootstrap. It SHALL reset the caret and refresh registry/list state after HTMX replacements.
- `/` SHALL focus the marked desktop search field. At the narrow responsive tier, where search is behind the primary-navigation disclosure, it SHALL open that disclosure instead. Escape SHALL close keyboard dialogs and the disclosure, clear the caret, and blur an editable control; all non-Escape shortcuts stand down in editable controls.
- Make the palette render sections immediately and filter them locally. It SHALL request escaped record rows from a feature-owned GET endpoint only after two characters and a 140 ms debounce; the request carries the remaining result capacity. The endpoint caps and orders public results, and its in-memory limiter periodically removes expired client windows.
- Use native dialogs for the palette and help, standard links for all destinations, a non-focus-stealing caret marker, and an initially hidden non-announcing hint bar. The layer respects reduced motion and does not display keyboard hints on touch-only devices.

## Risks / Trade-offs

- [Shortcut collisions] → reserve movement keys, restrict section bindings to the generated registry, and ignore editable contexts other than Escape and the command-palette shortcut.
- [Suggestion endpoint abuse] → minimum query length, debounce, remaining-capacity requests, capped results, rate limiting, and expiry cleanup.
- [HTMX DOM replacement leaves stale caret state] → refresh registry/list state and drop the caret after the existing settle lifecycle.
- [Responsive search differs by tier] → mark both the desktop search control and the mobile navigation disclosure explicitly, then cover both behaviours in browser tests.
