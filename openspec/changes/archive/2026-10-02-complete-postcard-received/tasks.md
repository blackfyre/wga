## 1. Received postcard

- [x] 1.1 Move the artwork location and dimensions derivation into the artwork package, reuse it on the artwork record page, and add a postcard-workflow projection of the received card with canonical record and compose URLs; verify `go test ./internal/artworks ./internal/postcards ./internal/handlers/artists ./internal/handlers/postcards`.
- [x] 1.2 Render the linked title, `VIEW IN GALLERY →`, `SEND YOUR OWN →`, `BROWSE THE ARCHIVE →`, and the optional dimensions and location lines as plain anchors; run `templ generate` and verify `go test ./internal/assets/templ/pages -run Postcard`.
- [x] 1.3 Align the composer character-count copy with the design in the template and `resources/js/postcard-rich-text.ts`; run `bun run build` and the scoped Biome check.

## 2. Verification

- [x] 2.1 Add Playwright coverage that follows `VIEW IN GALLERY →` to the selected work and `SEND YOUR OWN →` to its composer, at 390px, 834px and 1440px and without JavaScript; run `mise run test:playwright --port 8892` on the postcard specs with Mailpit, then the full suite.
- [x] 2.2 Run `go vet ./...`, `go test ./... -cover` and `golangci-lint run`.
