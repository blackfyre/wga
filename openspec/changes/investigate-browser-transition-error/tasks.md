## 1. Attribute and reproduce

- [x] Read staging Sentry issue and event; record breadcrumb attribution and absent source-mapped stack in design.md.
- [x] Reproduce native ready InvalidStateError through artwork search; Playwright fails without the fix with two snapshot-capture errors.

## 2. Guard and verify

- [x] Add a narrow native ready guard preserving callback errors and fallback; all 6 browser-transition tests pass on port 8108, including two requests and exactly two swaps in the native capture regression. All 21 existing artwork-search tests also pass.
- [x] Run go mod tidy (no module diff), templ generate, bun run build, go vet ./..., go test ./... -cover, golangci-lint run, relevant Playwright specs on port 8108, and format changed Markdown/JS.

## 3. Deliver

- [ ] Archive the change, format synced specs, and pass openspec validate --specs.
- [ ] Push and open the PR with attribution limit and sequence diagram; CI ci-gate passes. Independent cross-family review is owned by the coordinator.
