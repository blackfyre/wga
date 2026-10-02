## 1. Home composition

- [x] 1.1 In `home.templ`, reduce the hero to BROWSE ARTISTS → (`/artists`) and COMPARE TWO WORKS → (`/dual-mode`); caption the work of the day with the capitalised artist, the date and a trailing arrow; relabel the counts REPRODUCTIONS, ARTISTS, SCHOOLS and CENTURY with the period "3rd–early 20th"; and replace the feature cards with TWO WINDOWS, TIMELINE and POSTCARD SERVICE (including MEET THE CONTRIBUTORS →). Verify with `templ generate`, `bun run build`, and an updated `go test ./internal/assets/templ/pages -run Home`.

## 2. Browser verification

- [x] 2.1 Update `playwright-tests/home.spec.ts` to assert the two hero actions and their hrefs, the three cards with their links to `/dual-mode`, `/timeline`, `/postcard` and `/contributors`, the reachable destinations, and no horizontal overflow at 390, 834 and 1440px. Verify with `mise run test:playwright --port 8982` on the home spec, then the full suite.

## 3. Final checks

- [x] 3.1 Run `go vet ./...`, `go test ./... -cover`, and `mise run check` with no new issues.
