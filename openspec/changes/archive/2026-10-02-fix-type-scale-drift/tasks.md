# Tasks

## 1. Prose and link typography

- [x] 1.1 Rewrite `.content` in design tokens inside `@layer components` (`--t-16`, 1.7, `--wga-text`; accent links with a 1px underline, inherited weight and secondary-accent hover; scale rungs for headings and code; disc and decimal lists) and move the `a:not([class])` default into `@layer base`; verify with `go test ./internal/assets -run '^TestProseRuleUsesDesignScaleInsideComponentsLayer$'` and `bun run build`.
- [x] 1.2 Add `playwright-tests/prose-typography.spec.ts` asserting computed font size, line height, text colour and link colour, decoration and weight on an artist biography, an artwork commentary and a static page at 390, 834 and 1440px in light and dark; verify with `mise run test:playwright --port 8890 playwright-tests/prose-typography.spec.ts`.

## 2. Off-scale sizes

- [x] 2.1 Map both `--t-24` uses and all 21 `text-lg`, `text-xl` and `text-2xl` template uses to the design's rung for each element; verify with `go test ./internal/assets -run '^TestPublicSourcesStayOnTheTypeScale$'`, `templ generate` and `bun run build`.

## 3. Redundant rules

- [x] 3.1 Replace the per-palette focus-ring hex overrides with `--wga-accent` in the base focus rule after confirming each equals that palette's token, and remove the unused `.home-prose` rule; verify with `go test ./internal/assets -run '^TestFocusRingAndDeadProseRulesAreNotDuplicated$'`.

## 4. Verification

- [x] 4.1 Run `go vet ./...`, `go test ./... -cover`, `golangci-lint run`, `bunx biome check` on changed TypeScript, and the full `mise run test:playwright --port 8890` suite; compare artist and artwork screenshots with the standalone design build.
