## 1. Permanent keyboard bar in the measured stack

- [x] 1.1 Show `.wga-kbd-bar` under `(hover: hover) and (pointer: fine)` without `html[data-kbd-on]`, remove `data-kbd-on` and `markUsed()` from `keyboard.ts`, and register the bar as bottom-stack item `keyboard` with order 0. Verify with `templ generate`, `bun run build`, a focused `go test ./internal/assets/templ/components`, and `bunx biome check resources/js/keyboard.ts`.

## 2. Floating controls clear the stack

- [x] 2.1 Add `--wga-floating-gap` (16px below 45rem, 24px from 45rem), move FEEDBACK's inline offset into the stylesheet, use the gap for `.wga-toast-stack`, add the bar's 30px to the pre-measurement stack-height fallback on desktop pointers, and move the `wga-bottom-stack-content` reservation from `<main>` to the shared layout's `<body>` so it follows the footer. Verify with `bun run build` and `go test ./internal/assets/templ/layouts`.

## 3. Browser verification

- [x] 3.1 Update `bottom-stack.spec.ts`, `feedback.spec.ts`, and `keyboard-navigation.spec.ts` to cover the bar before any key press on desktop and its absence on touch, and FEEDBACK and toast offsets of 24px/16px above the highest docked surface with no tray, the itinerary tray, the Study Board tray, and both, at 390, 834, and 1440px, plus final-content reachability. Verify with `mise run test:playwright --port 8970` on those specs, then the full suite.

## 4. Records and final checks

- [x] 4.1 Update `docs/design-sync.md` to record the 2 October 2026 entry as last implemented, and run `go vet ./...`, `go test ./... -cover`, and `golangci-lint run` with no new issues.
