## 1. Exact artist catalogue state

- [x] 1.1 Extend the artwork-search filter state, canonical URL generation, and persistence condition with exact `artist_id` membership and legacy-name precedence; verify focused artwork-search tests cover duplicate/overlapping filing names, co-authored works, unknown IDs, and mixed-parameter canonicalisation.
- [x] 1.2 Preserve active `artist_id` as non-visible GET state and through result view, sorting, pagination, reset, and Dual Mode URLs without exposing an artist-ID form control; verify focused page and handler tests cover enhanced and ordinary GET submissions.

## 2. Artist-record hand-off

- [x] 2.1 Change the artist-record `FIND MORE BY … IN THE ARTWORK SEARCH` URL to pass the public artist record ID while retaining all other artist-derived holding links; verify artist-record view and template tests assert the exact generated URL and unchanged out-of-scope links.

## 3. Integration verification

- [x] 3.1 Add end-to-end handler coverage for full-page and HTMX artist-holding requests, including retained exact state after a catalogue refinement; run `templ generate`, the affected Go test packages, and `go vet ./...` successfully.
