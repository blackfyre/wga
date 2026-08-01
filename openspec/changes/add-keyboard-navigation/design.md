## Context

The reference keyboard layer combines discoverable help, section jumps, caret navigation, and a command palette. WGA currently initialises public browser helpers in `bootstrap.ts` but has no suggestion endpoint or common list-markup contract.

## Goals / Non-Goals

**Goals:**

- Provide accessible, scoped shortcuts without intercepting form editing.
- Add a rate-limited, server-rendered record suggestion endpoint for the palette.

**Non-Goals:**

- Replace browser navigation, add client-side routing, or add timeline/itinerary shortcuts.

## Decisions

- Render the keyboard layer once from the public layout; pages opt in with data attributes for navigable lists and search fields.
- Keep one authoritative keyboard module registered from the existing browser bootstrap and reset state after HTMX swaps.
- Make the palette query a feature-owned GET suggestion route returning escaped HTML rows; enforce minimum query length and rate limits.
- Ignore shortcuts while focus is in editable controls and expose help, focus management, Escape dismissal, and standard links for accessibility.

## Risks / Trade-offs

- [Shortcut collisions] → restrict bindings to unmodified keys and ignore editable contexts.
- [Suggestion endpoint abuse] → minimum query length, capped results, and rate limiting.
- [HTMX DOM replacement leaves stale focus] → listen for the existing settle lifecycle and reset caret state.
