## Why

The public archive has deep catalogue navigation but no keyboard-first path through sections, results, or record lookup. The updated reference defines an accessible keyboard layer and command palette.

## What Changes

- Add global keyboard shortcuts, section jumps, result-list caret navigation, shortcuts help, and a command palette.
- Add a rate-limited server-rendered suggestion endpoint for the palette.
- Add layout and page markup contracts required by the shared keyboard layer.

## Capabilities

### New Capabilities

- `keyboard-navigation`: Accessible keyboard navigation, command-palette lookup, and keyboard help across public pages.

### Modified Capabilities

- None.

## Impact

- Public layout, browser bundle, public page markup, a new search-suggestion handler endpoint, accessibility behaviour, and focused Go/Playwright coverage.
