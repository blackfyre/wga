# Proposal

## Why

The text-driven filter forms on the artists, artworks, Dual Mode index, guestbook, and glossary pages appear to search on every character. Each form lives inside the block its own response replaces, so every result rebuilds the field being typed into: the server echoes a value that may already be stale, the pending debounce is discarded with the old form, and the short 300 ms delay fires between ordinary keystrokes. Visitors see repeated re-renders and risk losing characters typed while a request is in flight.

## What Changes

- Free-text search fields on the five affected forms keep the visitor's in-progress text, focus, and caret when a result produced by that same form is swapped in.
- Results from any other interaction on the same block (reset, sort, view, letter or pane navigation, history restore, full page load) render the server's value for those fields, so reset and navigation still clear or replace the typed text.
- A newer search from the same form supersedes one still in flight on every affected form, so stale results cannot overwrite newer ones.
- Text-driven search waits for a longer pause in typing (500 ms, from 300 ms) on every public text-search form, including global search, and ignores key events that do not change the field's value.
- Out of scope: range sliders (born-year and timeline windows), which are rebuilt in the same way but involve a different interaction and server-side value clamping; restructuring any form so that it sits outside its swap target; and changes to search semantics or results.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: adds a shared requirement for how enhanced public text-search forms debounce, order, and preserve in-progress input across their own result swaps.

## Impact

- Templ pages and components: `artworks.templ`, `artists.templ`, `dual.templ`, `guestbook.templ`, `glossary.templ`, `search.templ` (delay only), and the shared `components.Field` with its `dto.Field`.
- Handlers for artworks, artists, Dual Mode, guestbook, and glossary read the request's `HX-Trigger` header to decide whether the originating form's text fields are preserved.
- Go template and handler tests. The existing trigger assertion in `internal/assets/templ/pages/artworks_test.go` changes.
- Playwright specs for the affected pages.
- No routes, persisted data, configuration, or dependencies change.
