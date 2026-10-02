# Tasks

## 1. Header layout

- [x] 1.1 In `TopNav()`, keep SEARCH and `CTRL K` at their natural width, let the search label fill the remaining space with a 140px minimum inside a form capped at 340px, use the design's 16px logo-row gap and an 8px form gap below 1080px; verify with `go test ./internal/assets/templ/components -run '^TestTopNav'`, `templ generate` and `bun run build`.
- [x] 1.2 Tighten the navigation row's item gap and the gap before MORE to 20px below 1080px so the row is one line at 834px; verify with the header browser check.

## 2. Brand and MORE

- [x] 2.1 Align the desktop wordmark and strapline type with the design, change the shared strapline text to lowercase ordinals, and render `MORE ▾` without the native marker; verify with `go test ./internal/assets/templ/components -run '^TestTopNav'`.

## 3. Verification

- [x] 3.1 Add `playwright-tests/header-layout.spec.ts` asserting the keyboard cue, field width, navigation line count, overflow, brand type and MORE marker at 720, 834, 1079, 1080 and 1440px, and update the affected header assertions in `public-navigation.spec.ts` and `public-shell.spec.ts`; verify with `mise run test:playwright --port 8971` on the new and affected specs.
- [x] 3.2 Run `go vet ./...`, `go test ./... -cover`, `golangci-lint run` and the full `mise run test:playwright --port 8971` suite; capture before and after screenshots at 834 and 1440px.
