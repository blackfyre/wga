## 1. Collection schema and migration

- [x] 1.1 Add PocketBase migrations for canonical subjects, series, and techniques collections, including the series-or-altarpiece distinction, plus location-alias storage; current museums and original locations reuse the producer-owned `locations` collection (`museum` and `is_public` flags) rather than a separate museums collection. Evidence: `internal/migrations/1784808002_add_relationship_paths.go`, `TestRelationshipPathsMigrationDefinesCollectionsAndArtworkRelations`.
- [x] 1.2 Add the replacement artwork relations for primary author, co-authors, subjects, series, original locations, and techniques; reuse `current_location_id` as the current-museum relation, `art_period_id` as the art-period relation, and the existing school relation for school/workshop paths. Evidence: same migration and test.
- [x] 1.3 Implement a non-destructive legacy-author migration that copies single assignments, reports multi-author records, and retains legacy data until review is complete. Evidence: `internal/migrations/1794028172_copy_legacy_artwork_authors.go` (logged review report plus the dashboard filter `primary_author = '' && author:length > 1`).
- [x] 1.4 Add migration tests covering collection definitions, a single legacy author, and a multi-author review case. Evidence: `internal/migrations/relationship_paths_test.go`.

## 2. Import and location normalisation

Location normalisation, private-collection classification, alias review, the unresolved-location report, and reviewed alias seeds are owned by the `wga-src` producer (its archived change `2026-07-31-add-artwork-connection-paths` and its `location-normalization` specification). WGA imports the producer's results and does not duplicate the normaliser or report.

- [x] 2.1 Extend the source-to-application import contract to retain raw artwork location text and populate the new canonical artwork relations: `artworks.location` → `source_location`, `artworks.primary_author_id` → `primary_author` (falling back to `author_id` for older exports), `techniques` and `artwork_techniques` → `techniques`, and `location_aliases` → `location_aliases`. No `wga-src` change was needed. Evidence: `internal/utils/seed/source.go`, `internal/utils/seed/database.go`, `TestRelationshipPathsImportContract`.
- [x] 2.2 Implement comparison-key normalisation and configurable private-collection variant classification without creating a public location relation. Evidence: `wga-src` `internal/importer/location_normalization.go` and its tests; WGA keeps the producer's non-public classification (`TestRelationshipPathsImportContract`) and never derives a public path from it (`TestRelatedByCollectionEmptyWhenPrivate`).
- [x] 2.3 Add canonical museum and alias storage, resolving exact normalised aliases to current-museum relations. Evidence: `wga-src` `internal/importer/reference_location.go` resolves exact aliases; WGA stores them in `location_aliases` and imports the resolved `current_location_id` (`TestRelationshipPathsImportContract`).
- [x] 2.4 Generate an unresolved-location report grouped by normalised value, with occurrence counts and scored edit-distance candidates for review only. Evidence: `wga-src` `internal/importer/location_normalization.go` report generation and its tests.
- [x] 2.5 Seed the initial reviewed museum aliases and add focused tests for raw-value preservation, private-collection exclusion, exact aliases, ambiguous candidates, and grouped reports. Evidence: `wga-src` reviewed alias batches and importer tests; WGA tests cover raw-value preservation, private-collection exclusion, and alias import.

## 3. Public relationship paths

- [x] 3.1 Add a relationship-query workflow for the three public bases: primary/co-author, current museum, and forty-year cross-artist date proximity. Evidence: `internal/repositories/artwork_related.go`, `TestRelatedBasesUsePrimaryAndCoAuthors` and the existing resolver tests.
- [x] 3.2 Update artwork detail DTOs, templates, and handlers to render the three-basis relation row, shareable active-basis state, and honest sparse-result treatment. Evidence: existing `internal/assets/templ/pages/artwork.templ`, `internal/handlers/artists/artworks.go`, and their tests.
- [x] 3.3 Replace application reads of the legacy `author` relation in artwork search, artist pages, artwork routes, URL generation, and JSON-LD with the primary-author and co-author relations, reading the legacy relation only while `primary_author` is empty until section 4 completes. Evidence: `internal/artworks/authors.go`, `internal/artworks/authors_test.go`, `TestBuildArtworkSearchViewUsesPrimaryCoAuthorAndLegacyFallback`.
- [x] 3.4 Add handler and template tests for all three public bases, their ordering/exclusions, shareable query state, and sparse-result alternatives. Evidence: existing `internal/handlers/artists/artwork_route_test.go`, `artworks_test.go`, `artwork_relationships_task73_test.go`, and `internal/assets/templ/pages/artwork_test.go`.
- [x] 3.5 Add a regression test that private-collection classifications produce no public related-artwork path. Evidence: `TestRelatedByCollectionEmptyWhenPrivate` and `TestRelatedHoldingCollectionCountAndPrivateNil`.

## 4. Verification and rollout

- [ ] 4.1 Run the import against representative data, review the legacy-author and unresolved-location reports, and curate approved aliases before enabling public museum paths.
- [ ] 4.2 Run `go vet ./...` and `go test ./... -cover` from the WGA repository.
- [ ] 4.3 Perform a manual PocketBase-admin check that artwork editors can assign the new relations without a custom relationship editor.
- [ ] 4.4 Verify public artwork pages, artist pages, search, canonical URLs, and JSON-LD after migration; remove the legacy `author` relation only after the review reports are clear.
