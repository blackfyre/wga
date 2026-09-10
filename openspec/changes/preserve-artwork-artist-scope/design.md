## Context

See proposal.md for motivation and the catalogue-exploration delta for observable behaviour. Exact artist scope is already represented by `artist_id` in filter state, canonical URL builders, and a hidden GET form control. The current Go integration coverage constructs refinement URLs directly, so it proves server handling but not the browser's form-to-HTMX parameter contract. The page-owned artwork-search view carries only the identifier and therefore cannot present the artist's name.

Artwork-search form interactions replace the complete `#artwork-search` block, while pagination can request only the results fragment. The results-only path deliberately avoids loading filter and page projections, so artist-scope presentation must not add work to that response.

## Goals / Non-Goals

**Goals:**

- Treat the hidden `artist_id`, HTMX form submission, handler parsing, canonical URL, replacement block, and resulting DOM as one verified interaction contract.
- Resolve a public artist scope independently from the current result page so its name remains visible when additional filters yield no works.
- Preserve the results-only response's bounded data-work contract.

**Non-Goals:**

- Do not change the meaning or public shape of `artist_id`, `artist`, or `q`.
- Do not make the scope independently removable; the existing reset action remains the way to clear it and all other filters.
- Do not reveal unpublished artist details or infer a name from matching artwork records.

## Decisions

### Present exact artist scope separately from the query

The page-owned search view will carry an optional resolved artist-scope presentation using the public artist's established filing-name identity. The filter rail will render that scope separately from the editable title-or-artist query. `artist_id` remains the submitted and canonical value; the displayed name is presentation only.

Alternative considered: put the artist name into the read-only `q` field and let `artist_id` supersede it. Rejected because it makes a displayed query value semantically false, prevents further text refinement, and couples exact relationship state to free-text search.

### Resolve only published artist identity on complete search views

When `artist_id` is active, complete page and search-block construction will use the existing public artist lookup boundary to resolve the filing name. A missing or unpublished artist will leave the optional presentation empty while the exact identifier continues through normal filtering and canonicalisation, preserving the honest no-results behaviour.

Results-only fragment construction will not perform this lookup because it neither renders nor swaps the filter rail. This preserves the catalogue requirement that results-only requests avoid page-wide projection work.

Alternative considered: derive the name from the returned artwork page. Rejected because a refined or out-of-range result page can be empty and a co-authored result's primary displayed artist need not be the scoped artist.

### Prove preservation at the browser boundary

A Playwright scenario will enter the search from an artist record, assert the hidden identifier and visible filing-name scope, refine through the live form, and assert that the outgoing/resulting URL and replaced DOM retain the exact scope. Focused Go tests will continue to cover view construction and template output, but will not substitute for the browser assertion.

The implementation should first reproduce the reported interaction with this scenario. Any preservation correction will remain at the form/HTMX contract boundary established by that evidence rather than introducing client-side URL reconstruction.

## Risks / Trade-offs

- [The existing source already appears to serialise `artist_id`, so the reported failure may depend on browser interaction or deployed assets] -> Establish the failure with Playwright and change only the demonstrated contract break; retain the regression even if the code correction is minimal.
- [Artist identity lookup adds work to artwork search] -> Perform it only when `artist_id` is present and only for responses that render the complete search block.
- [An arbitrary identifier could expose unpublished artist data] -> Use the existing published-artist lookup contract and render no artist details when it does not resolve.
- [The visible label could be mistaken for editable text] -> Present it as scope metadata outside the query control and keep `q` visibly editable.

## Migration Plan

No data or URL migration is required. Deploy the server and generated frontend assets together. Rollback removes the scope presentation and regression-specific correction while existing `artist_id` URLs retain their established server semantics.
