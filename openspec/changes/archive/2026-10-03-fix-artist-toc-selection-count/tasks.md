# Tasks

## 1. Selection count

- [x] 1.1 Render each `ON THIS PAGE` selection entry from the selection's work count, with `1 work` for a single work. Verify with `go test ./internal/assets/templ/pages -run TOC`, covering a seven-work selection with a four-work preview and a one-work selection.
- [x] 1.2 Confirm that the existing selection Playwright spec still passes: `mise run test:playwright playwright-tests/selection.spec.ts`.
