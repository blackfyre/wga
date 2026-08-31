## 1. Sitemap workflow and durable publication

- [x] 1.1 Refactor sitemap generation to produce an uncompressed canonical index and matching child maps from a dedicated `app.DataDir()` directory, including only valid published artist and artwork URLs; verify focused generation tests parse the index and child XML.
- [x] 1.2 Stage, validate, atomically publish, and prune generated sitemap sets while preserving the prior complete set on failure; verify tests cover failed regeneration, stale-file cleanup, and non-terminating error handling.
- [x] 1.3 Run the shared sitemap workflow after application readiness, on the daily cron schedule, and from the existing manual command; verify lifecycle tests assert initial invocation, scheduled failure survival, and structured outcome logging.

## 2. Public sitemap delivery and discovery

- [x] 2.1 Register canonical index, named child-map wildcard, and `robots.txt` routes backed by the data-directory sitemap set; verify full-app route tests retrieve the index, each referenced child map, and the absolute discovery directive.
- [x] 2.2 Add a same-origin sitemap XSL response and stylesheet processing instructions that load the current site CSS without JavaScript; verify XML tests retain crawler-readable data and browser coverage confirms a styled sitemap view.

## 3. Integration verification

- [x] 3.1 Run the focused sitemap, static-handler, cron, and entrypoint Go tests; verify canonical URL inclusion/exclusion, publication failure retention, route boundaries, and lifecycle behaviour pass.
- [x] 3.2 Run `go vet ./...` and `go test ./... -cover`; verify the wider static analysis and Go suite pass after the cross-component change.
- [x] 3.3 Perform an independent review of the generator-to-public-route contract after deterministic checks pass; address valid findings and rerun affected checks.
