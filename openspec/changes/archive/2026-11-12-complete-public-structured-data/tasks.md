## 1. Structured data and descriptions

- [x] 1.1 Add the brand logo to the publisher `Organization`; verify with `go test ./internal/assets/templ/utils -run '^TestSiteJSONLDDescribesWebsiteAndPublisher$'`.
- [x] 1.2 Describe selections as an `ItemList` of their shown works; verify with `go test ./internal/utils/jsonld -run '^TestSelectionJsonLdListsShownWorksInOrder$'` and `go test ./internal/handlers/artists -run '^TestSelectionRouteRendersFullPage$'`.
- [x] 1.3 Produce plain-text, character-safe descriptions; verify with `go test ./internal/assets/templ/utils -run '^TestDescription'`.
- [x] 1.4 Remove the unreachable artist content renderer; verify with `golangci-lint run`.

## 2. Verification

- [x] 2.1 Run `go vet ./...`, `go test ./... -cover` and `golangci-lint run`.
- [x] 2.2 Validate the change with `openspec validate complete-public-structured-data --strict`.
