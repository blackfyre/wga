## 1. Collection schema and migration

- [ ] 1.1 Add PocketBase migrations for canonical subjects, series, locations, museums, and techniques collections, including the series-or-altarpiece distinction.
- [ ] 1.2 Add the replacement artwork relations for primary author, co-authors, subjects, series, original locations, current museums, techniques, and art periods; reuse the existing school relation for school/workshop paths.
- [ ] 1.3 Implement a non-destructive legacy-author migration that copies single assignments, reports multi-author records, and retains legacy data until review is complete.
- [ ] 1.4 Add migration tests covering collection definitions, a single legacy author, and a multi-author review case.

## 2. Import and location normalisation

- [ ] 2.1 Extend the source-to-application import contract to retain raw artwork location text and populate the new canonical artwork relations.
- [ ] 2.2 Implement comparison-key normalisation and configurable private-collection variant classification without creating a public location relation.
- [ ] 2.3 Add canonical museum and alias storage, resolving exact normalised aliases to current-museum relations.
- [ ] 2.4 Generate an unresolved-location report grouped by normalised value, with occurrence counts and scored edit-distance candidates for review only.
- [ ] 2.5 Seed the initial reviewed museum aliases and add focused tests for raw-value preservation, private-collection exclusion, exact aliases, ambiguous candidates, and grouped reports.

## 3. Public relationship paths

- [ ] 3.1 Add a relationship-query workflow that finds published artworks sharing each configured canonical relation, deduplicates artworks, and retains every shared reason.
- [ ] 3.2 Update artwork detail DTOs, templates, and handlers to render related artworks with their connection-path labels.
- [ ] 3.3 Replace application reads of the legacy `author` relation in artwork search, artist pages, artwork routes, URL generation, and JSON-LD with the primary-author and co-author relations.
- [ ] 3.4 Add handler and template tests for shared artist, school/workshop, subject, series/altarpiece, original location, current museum, technique, and period paths, including multiple paths for one artwork pair.
- [ ] 3.5 Add a regression test that private-collection classifications produce no public related-artwork path.

## 4. Verification and rollout

- [ ] 4.1 Run the import against representative data, review the legacy-author and unresolved-location reports, and curate approved aliases before enabling public museum paths.
- [ ] 4.2 Run `go vet ./...` and `go test ./... -cover` from the WGA repository.
- [ ] 4.3 Perform a manual PocketBase-admin check that artwork editors can assign the new relations without a custom relationship editor.
- [ ] 4.4 Verify public artwork pages, artist pages, search, canonical URLs, and JSON-LD after migration; remove the legacy `author` relation only after the review reports are clear.
