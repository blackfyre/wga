## 1. Structured data

- [x] 1.1 Rebuild the JSON-LD builders with typed, linked nodes, omitted empty values, block-aware plain text and breadcrumb lists; verify with `go test ./internal/utils/jsonld`.
- [x] 1.2 Describe artwork records from the visible view on the canonical URL; verify with `go test ./internal/handlers/artists -run '^TestArtworkRouteDescribesRecordWithCanonicalJSONLD$'`.
- [x] 1.3 Add breadcrumb JSON-LD to artist and selection pages and reduce index nodes to visible fields; verify with `go test ./internal/handlers/artists -run '^TestSelectionRouteRendersFullPage$'`.

## 2. Verification

- [x] 2.1 Run `go vet ./...`, `go test ./... -cover` and `golangci-lint run`.
- [x] 2.2 Validate the change with `openspec validate repair-record-structured-data --strict`.
