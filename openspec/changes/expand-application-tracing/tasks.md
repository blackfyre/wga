## 1. Shared Trace Primitives

- [x] 1.1 Extend the observability package with typed, allow-listed workflow and repository operation spans that record bounded success or failure outcomes, and verify focused tests reject unknown names and arbitrary error text.
- [x] 1.2 Add bounded cache outcome events to the active recording span while preserving existing cache metrics and no-op behaviour, and verify hit, miss, shared, failure, cache-name, and cache-key privacy cases in focused cache tests.

## 2. Catalogue Repository Detail

- [x] 2.1 Instrument context-aware artist-index count and list operations with the shared repository span helper, and verify focused repository tests cover nesting, stable names, outcomes, and context-free compatibility paths.

## 3. Catalogue Workflow Detail

- [x] 3.1 Instrument artwork-search result and facet stages with coarse workflow spans, and verify an integration trace contains the expected hierarchy without filter values, record identity, or duplicate loop-level spans.
- [x] 3.2 Instrument dual-mode reference loading, pane construction, and rendering with coarse workflow spans, and verify representative index and record-pane traces contain stable stage names without pane paths, record identity, or visitor-controlled state.

## 4. Integration Verification

- [x] 4.1 Run focused observability, cache, artist-index, artwork-search, and dual-mode tests followed by `go vet ./...`, `go test ./... -cover`, and strict OpenSpec validation; verify a configured local trace shows the bounded child-span and cache-event structure while disabled telemetry preserves existing behaviour.
