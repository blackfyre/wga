## Why

The public archive has a partial keyboard layer, but it does not meet the updated reference's keyboard-first path through sections, results, search, or record lookup. The existing change incorrectly records that reduced implementation as complete.

## What Changes

- Complete global section shortcuts, two-digit section jumps, responsive search access, and complete Escape dismissal.
- Add a generated screen registry shared by shortcuts, the help surface, the palette, and browser behaviour.
- Complete caret-based list and grid traversal across the supported public index screens.
- Make the command palette search sections and public records, with debounced, capped, rate-limited suggestions.
- Replace completed tasks with the outstanding implementation and verification work.

## Capabilities

### New Capabilities

- `keyboard-navigation`: Accessible keyboard navigation, command-palette lookup, and keyboard help across public pages.

### Modified Capabilities

- None.

## Impact

- Public layout, browser bundle, public page markup, the keyboard suggestion endpoint and limiter, accessibility behaviour, and focused Go/Playwright coverage.
