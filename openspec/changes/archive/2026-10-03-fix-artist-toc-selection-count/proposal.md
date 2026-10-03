# Proposal

## Why

On a curated artist record, every selection's `ON THIS PAGE` entry ends with "4 works", whatever the selection holds. The entry counts the selection's preview grid, which is capped at four works (`selectionPreviewWorkLimit`). The requirement asked for the "shown-work count", so it captured the cap rather than the selection's size, and visitors read it as the size of the selection.

## What Changes

- Each selection entry in `ON THIS PAGE` reports the number of works in that selection: the same figure as the selection's `N SELECTED` count. It uses `1 work` for a single work.
- The preview grid stays capped at four works.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artist-selections`: the in-page index requirement reports the selection's work count instead of the preview's.

## Impact

- `internal/assets/templ/pages/artist.templ` and its Go test.
