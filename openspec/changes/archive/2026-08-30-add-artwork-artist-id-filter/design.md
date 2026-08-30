## Context

See proposal.md for motivation and the catalogue-exploration delta for observable behaviour. The existing catalogue accepts a name-based `artist` URL value and performs a filing-name match. Artist-record routes already carry a stable public PocketBase record ID, but the catalogue does not consume it as filter state.

Artwork authors are multi-value relations. Catalogue state is constructed once and then reused to build database conditions, canonical URLs, pagination, result-view and sort links, and Dual Mode hand-off URLs. The catalogue form uses GET for both ordinary navigation and enhanced fragment updates, so URL-only state must also be represented as non-visible submitted state.

## Goals / Non-Goals

**Goals:**

- Make artist-record holding links select an exact artist relationship rather than a filing-name substring.
- Preserve the exact filter across every existing catalogue navigation path without introducing a visible artist-ID input.
- Retain compatibility for existing name-based `artist` URLs.

**Non-Goals:**

- Do not convert selection or artwork-related holding links to ID filtering.
- Do not add a schema migration, index, dependency, or claimed performance improvement.
- Do not replace the existing free-text artist search behaviour.

## Decisions

### Use `artist_id` as a public URL-state parameter

The parameter carries the existing public PocketBase artist record ID used by artist routes. It is parsed as catalogue filter state, serialised into canonical URLs and all derived catalogue links, and not rendered as a visible form control.

An exact relationship condition shall use the relation's multi-value membership semantics, so a work with the artist in any author position matches. This avoids the current filing-name substring ambiguity and avoids treating a co-author as absent.

Alternative considered: replace the existing `artist` parameter. Rejected because bookmarked name-based URLs already have observable behaviour. The old parameter remains supported for itself; if both appear, `artist_id` is authoritative and canonicalisation removes `artist` to avoid mismatched intersections.

### Preserve URL-only state through GET submission

The rendered catalogue form shall include `artist_id` only as non-visible preserved request state while it is active. This is required because both normal GET submission and enhanced form serialization otherwise discard a parameter that has no successful form control. It is not a user-selectable filter input.

Alternative considered: client-side URL reconstruction. Rejected because it would change no-JavaScript behaviour and duplicate server-owned URL-state logic.

### Update only the artist-record holding link

The artist-record view has the exact `FIND MORE BY … IN THE ARTWORK SEARCH` call to action specified by this change. Its URL builder receives the artist record ID rather than the filing name. Selection and related-work holding routes are explicitly unchanged, preventing the change from widening beyond the approved call to action.

### Retain existing public-result rules and empty-state behaviour

The exact filter composes with existing published-artwork and other catalogue predicates. It does not perform an additional artist lookup merely to validate the ID; an unrecognised or unmatched ID naturally produces the established empty result state.

## HTMX and progressive-navigation contract

```text
artist-record link
  -> GET /artworks?artist_id=<public-record-id>
  -> catalogue handler builds exact relation condition and canonical URL
  -> full page for ordinary GET, result fragment for enhanced request
  -> existing catalogue target swaps only the result block
  -> form, sort, view, pagination, and Dual Mode URLs retain artist_id
```

The visible filter form remains unchanged. The filter state is retained in the form as non-visible request data, so an enhanced swap and a JavaScript-disabled ordinary submission have equivalent URL and result semantics.

## Operational Decomposition

1. **Catalogue filter-state contract** — Area: the artwork-search filter, condition, canonical-URL, and derived-link workflow. Add exact-ID parsing, precedence, relation membership, and preserved state. This is the shared contract owner and must be completed before dependent tests.
2. **Artist-record link hand-off** — Area: artist-record view-model URL construction. Emit the exact artist ID in the existing call to action; do not modify other holding links. Depends on the catalogue contract.
3. **Acceptance coverage** — Area: catalogue and artist handler/template tests. Prove exactness, co-author inclusion, precedence, state propagation, reset, compatibility, HTMX and ordinary GET behaviour. Depends on both prior workstreams.

These workstreams are serialised around the shared catalogue filter contract. No schema, migration, cross-feature persistence helper, or external service coordination is required.

## Risks / Trade-offs

- **URL-only state is lost when a visitor refines filters** → retain active `artist_id` as non-visible GET state and test enhanced and JavaScript-disabled submission.
- **An exact filter excludes co-authored works** → use multi-relation membership and fixture coverage with the matching artist in a non-primary position.
- **Mixed legacy and exact parameters have ambiguous results** → give the exact parameter precedence and canonicalise away the legacy parameter.
- **Exact matching is described as a performance improvement without evidence** → make no performance claim; assess query plans separately if performance becomes a requirement.
- **Direct IDs reveal an unintended public boundary** → retain the existing published-artwork predicate and cover unknown/unmatched IDs with the normal empty state.

## Migration Plan

No data migration is required. Deploy as an additive URL-state capability:

1. Existing `artist` links and bookmarks continue to resolve with current semantics.
2. New artist-record links emit `artist_id`.
3. Rollback restores name-based link generation; no persisted data or URL schema needs repair.

## Verification Strategy

- Focused Go tests prove exact results for duplicate/overlapping filing names, co-author inclusion, unknown IDs, legacy compatibility, mixed-value canonicalisation, reset, pagination, sort/view, and Dual Mode propagation.
- Handler/template tests prove the artist-record link uses the record ID and no visible identifier form field exists.
- Exercise both an enhanced request and ordinary GET submission to prove the same retained URL state and result set.
- Run Templ generation after `.templ` changes, then the affected Go package tests and `go vet ./...` appropriate to the final diff.
