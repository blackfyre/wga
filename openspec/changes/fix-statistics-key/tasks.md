## 1. One swatch per key row

- [x] 1.1 Remove the legacy `.art-form-label::before` rules from `resources/css/style.pcss`. Verify with `bun run build` and a templ test that the key row renders one swatch.

## 2. Key as the single colour source

- [x] 2.1 Mark each art-form and school key swatch with the token it paints, and "Other" as a hatch, in `statistics.templ`. Verify with `templ generate` and `go test ./internal/assets/templ/pages -run '^TestStatistics'`.
- [x] 2.2 Resolve chart colours from the key in `resources/js/statistics.ts`: draw "Other" as a canvas hatch, switch off the canvas legend, expose `data-series-colours`, and redraw on palette changes. Verify with `bun run build` and `bunx biome check resources/js/statistics.ts`.

## 3. Copy

- [x] 3.1 Align the artworks subtitle, the table captions and the footnote with the design, keeping the truthful "RECOMPUTED HOURLY" footnote. Verify with the templ test.

## 4. Verification

- [x] 4.1 Add a Playwright check that each key row has one swatch matching the chart series colour, in light and dark and after a palette change. Run `mise run test:playwright --port <free port>` on the Statistics specs, then the full suite.
- [x] 4.2 Run `go vet ./...`, `go test ./... -cover` and `golangci-lint run`, and compare the page against the design screenshot.
