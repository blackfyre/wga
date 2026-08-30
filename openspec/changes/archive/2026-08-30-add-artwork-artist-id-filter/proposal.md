## Why

The artist-record link to the artwork search currently filters by an artist's filing name. Name matching can return works by differently identified artists with overlapping names, so the resulting holding is not reliably the artist whose record the visitor opened.

The link should use the artist record identifier already present in the public artist route to make the resulting catalogue holding exact, shareable, and stable across name presentation changes.

## What Changes

- Add an `artist_id` URL filter for artwork search that selects published artworks related to that exact artist record.
- Change the artist-record `FIND MORE BY … IN THE ARTWORK SEARCH` link to use `artist_id`.
- Preserve the ID filter through catalogue navigation, result presentation changes, sorting, pagination, Dual Mode hand-off, HTMX updates, ordinary GET submission, and reset.
- Keep the filter out of the visible catalogue form controls while retaining its state for subsequent filter changes.
- Retain existing name-based `artist` URLs for compatibility; when both values are supplied, the canonical state is defined by the exact ID filter.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalogue-exploration`: Artwork search gains an exact, URL-addressable artist-record filter and artist-record holding links use it.

## Impact

- Affects the artwork-search request/filter state, canonical URL generation, result links, and the artist-record view model.
- Affects server-rendered and HTMX catalogue navigation behaviour, including JavaScript-disabled GET submission.
- Does not require a schema change, new dependency, or visible filter control.

## Scope and non-goals

This change updates the artist-record `FIND MORE BY … IN THE ARTWORK SEARCH` link. Other artist-derived holding links keep their existing name-based behaviour unless separately approved.

The change does not promise a performance improvement or add an index; performance requires production-data query-plan evidence before any database work is proposed.

## Assumptions and risks

- `artist_id` denotes the existing public PocketBase artist record ID, not a new producer-facing identifier.
- A hidden state input is not a visible form control and is necessary to preserve a URL-only filter across enhanced and ordinary GET submissions.
- An unknown ID returns the existing honest empty state without a separate artist lookup.
- Exact relation membership must include co-authored artworks. The implementation must not assume the artist is first in the author list.
