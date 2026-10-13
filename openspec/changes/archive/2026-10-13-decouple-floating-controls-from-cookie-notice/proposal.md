# Proposal

## Why

The cookie notice is a bottom-left box, but FEEDBACK and the toast container are positioned above the total measured bottom stack, which includes the notice. On desktop widths, FEEDBACK therefore jumps up by the notice's height even though the two never share a column.

## What Changes

- FEEDBACK and the toast container clear the docked surfaces (keyboard hint bar, itinerary tray, Study Board tray) and clear the cookie notice only when it overlaps them horizontally, as it does at phone widths where the notice spans the viewport.
- The page-content reservation still includes the notice.

## Capabilities

### Modified Capabilities

- `public-page-experience`: the fixed bottom-stack requirement distinguishes docked surfaces from the floating cookie notice.

## Impact

`resources/js/bottom-stack.ts`, `resources/css/style.pcss`, and `playwright-tests/bottom-stack.spec.ts`.
