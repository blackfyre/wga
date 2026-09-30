# Tasks

## 1. Admission budgets

- [x] 1.1 Add `WGA_ITINERARY_DRAFT_BUDGET` and `WGA_ITINERARY_PUBLISH_BUDGET` to `internal/config`, defaulting to 3 and failing validation on non-positive or non-integer values, and document them in `.env.example`. Verify with focused config tests (`go test ./internal/config -run 'Itinerary'`).
- [x] 1.2 Construct the itinerary admission limiter from the configured budgets. Verify with `go test ./internal/itineraries/... ./internal/handlers/itineraries/...`, including a test that a fourth draft is refused under defaults.
- [x] 1.3 Set raised budgets in `mise.toml` `test:playwright` and in the CI Playwright job's application environment, and update the configuration section of `docs/development-guide.md`. Verify with `mise run test:playwright playwright-tests/itinerary.spec.ts playwright-tests/itinerary-task92.spec.ts playwright-tests/study-board.spec.ts playwright-tests/public-shell.spec.ts`.

## 2. Visual-overhaul drift

- [ ] 2.1 Restore `data-viewer-no-navbar` on the artwork record plate's viewer trigger and add a Templ render assertion. Verify with `templ generate`, `go test ./internal/assets/templ/pages -run 'Artwork'`, and `mise run test:playwright playwright-tests/artwork.spec.ts`.
- [ ] 2.2 Remove the file-weight expectation from `artwork.spec.ts`, as the catalogue-exploration spec requires. Verify with `mise run test:playwright playwright-tests/artwork.spec.ts`.
- [ ] 2.3 Compare the mobile logo mark and the large-viewport footer columns with `internal/assets/reference/visual-overhaul.html`. Update `public-navigation.spec.ts:285` and `public-shell.spec.ts:155` where the markup matches the reference, or fix the markup where it doesn't. Verify with `mise run test:playwright playwright-tests/public-navigation.spec.ts playwright-tests/public-shell.spec.ts`.

## 3. No-JavaScript stability

- [ ] 3.1 Diagnose the shared "element is not stable" cause in `guestbook:94`, `artwork-search:374`, `timeline-task81:105`, and `reference-pages:115` from their Playwright traces, and fix it at its source, not with forced clicks or longer timeouts. Verify with `mise run test:playwright` limited to those four spec files.

## 4. Layout and interaction

- [ ] 4.1 Fix the artist-record horizontal overflow at 200% text. Verify with `mise run test:playwright playwright-tests/artist-record.spec.ts`.
- [ ] 4.2 Fix the cookie-notice overflow at 390 px. Verify with `mise run test:playwright playwright-tests/cookieconsent.spec.ts`.
- [ ] 4.3 Establish why `keyboard-navigation.spec.ts:512` never observes `/artworks/results` with `q=Synthetic Artwork 01-01`. Fix the application if the search request regressed, otherwise align the spec with the current request contract. Verify with `mise run test:playwright playwright-tests/keyboard-navigation.spec.ts`.

## 5. Integration

- [ ] 5.1 Run `go vet ./...`, `go test ./... -cover`, `bun run build`, and the full `mise run test:playwright`. Confirm that the only remaining failure is postcard's Mailpit dependency when Mailpit is not running.
