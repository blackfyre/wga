# Tasks

## 1. Remove the embedded reference

- [x] 1.1 Delete `internal/assets/reference/visual-overhaul.html` and `internal/assets/reference.go`, the `/tmp/visual-overhaul` and `/tmp/visual-overhaul/footer` routes, the footer fixture builder, and the `/tmp/visual-overhaul` boundary entries in the static handler and trusted head markup; remove `visual_overhaul_test.go` and the related test cases; verify with `go vet ./...` and `go test ./internal/handlers/... ./internal/assets/...`.
- [x] 1.2 Re-point the `preferences.spec.ts` footer-replacement coverage at a real public page and update the `public-navigation.spec.ts` comment; verify with `mise run test:playwright --port 8881 playwright-tests/preferences.spec.ts playwright-tests/public-navigation.spec.ts`.
- [x] 1.3 Remove the reference file exclusion from `.github/workflows/codeql.yml`; verify with `bunx prettier --check .github/workflows/codeql.yml`.
- [x] 1.4 Confirm that no reference to the removed file or routes remains outside `openspec/changes/archive/`, using `grep -rn`, and compare `go build` binary sizes before and after.

## 2. Design intake

- [ ] 2.1 Add `docs/design-sync.md` recording the design project, the authoritative file, the last implemented entry (21 September 2026), the in-progress 29 September 2026 entries, and how to read the design; verify with `bunx prettier --check docs/design-sync.md`.
- [ ] 2.2 Add the `design-intake` skill in `.agents/skills/`, retire `visual-overhaul-reference-parity`, and update `AGENTS.md`; verify with `bunx prettier --check` on the changed Markdown.

## 3. Integration

- [ ] 3.1 Run `go vet ./...`, `go test ./... -cover`, `golangci-lint run`, the full `mise run test:playwright --port 8881`, and `openspec validate adopt-direct-design-intake`.
