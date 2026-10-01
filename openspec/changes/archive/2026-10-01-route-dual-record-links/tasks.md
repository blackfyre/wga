# Tasks

## 1. Pane-routed embedded links

- [x] 1.1 Rewrite supported record links in Dual Mode biography, selection-preview commentary, and selection-record commentary HTML to routed `/dual-mode` URLs with the pane-link HTMX attributes (target/select = routed pane, `outerHTML` swap), leaving other links unchanged and applying the rewrite after glossary annotation; verify with focused tests in `internal/handlers/dual` for opposite-pane, same-pane, selection-commentary, external, and non-record links (`go test ./internal/handlers/dual -run 'EmbeddedLink'`).
- [x] 1.2 Add a `playwright-tests/dual-mode.spec.ts` journey using a synthetic artist whose biography links to another artist: with JavaScript, the link loads into the routed pane and the other pane is unchanged; without JavaScript, the link navigates to the equivalent `/dual-mode` URL; verify with `mise run test:playwright playwright-tests/dual-mode.spec.ts`.
- [x] 1.3 Add the embedded-link acceptance criterion to `docs/features/dual-mode.md` per `docs/documentation-maintenance.md`; then run `go vet ./...` and `go test ./... -cover`.
