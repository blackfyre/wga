## 1. Site-level metadata

- [x] 1.1 Build the site-level `WebSite` and `Organization` JSON-LD graph from the configured public URL, serialised with a JSON encoder that escapes `<`; verify with `go test ./internal/assets/templ/utils -run '^TestSiteJSONLD'`.
- [x] 1.2 Add `og:locale` `en_GB` to the Open Graph tags; verify with `go test ./internal/assets/templ/utils -run '^TestMetadataDefaultsAreComplete$'`.
- [x] 1.3 Render the site-level block in the shared layout `<head>`; run `templ generate` and verify with `go test ./internal/assets/templ/layouts`.

## 2. Verification

- [x] 2.1 Run `go vet ./...`, `go test ./... -cover` and `golangci-lint run`.
- [x] 2.2 Validate the change with `openspec validate add-site-structured-data --strict`.
