## 1. Dual Mode Reference Projection

- [x] 1.1 Add a repository-owned, context-aware Dual Mode reference projection using the existing coalesced cache with an allow-listed telemetry identity, and verify focused tests cover one cold load, cache hits, concurrent sharing, defensive result isolation, loader failure, and cancellation.
- [x] 1.2 Add successful artist, school, and art-period mutation hooks that invalidate the Dual Mode reference generation, and verify hook tests cover create, update, delete, and invalidation during an in-flight load without restoring stale data.
- [x] 1.3 Replace Dual Mode's per-request reference queries with the repository projection while preserving local lookup construction, cancellation checkpoints, and context-free compatibility helpers; verify focused route tests preserve index, record-pane, canonical URL, and rendering behaviour while repeated requests avoid reference reloads.

## 2. Artwork Facet Diagnostics

- [x] 2.1 Extend the typed workflow allow-list with artwork option, venue, school-count, and form-count stages, and verify focused observability tests reject unknown stages and arbitrary error text.
- [x] 2.2 Instrument full artwork-search facet construction with the four child stages under the existing facet workflow span, and verify trace integration tests cover hierarchy, stable single occurrences, cancellation propagation, results-only omission, and exclusion of filters, identifiers, SQL, and result content.

## 3. Verification

- [x] 3.1 Run focused repository, hook, Dual Mode, artwork-search, cache, and observability tests followed by `go vet ./...`, `go test ./... -cover`, and strict OpenSpec validation; verify configured local traces expose the refined stages while disabled telemetry and context-free paths preserve existing behaviour.
