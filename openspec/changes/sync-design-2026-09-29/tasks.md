## 1. Remembered artwork search presentation (server)

- [x] 1.1 Parse the `wga_aw_prefs` cookie in the artworks package, apply its sort, direction, and view only to a bare `GET /artworks`, expose its actions flag to the results view and layout, and add `Vary: Cookie`. Verify with focused `go test ./internal/handlers/artworks -run 'SearchPrefs'` covering explicit parameters winning, reset keeping the preferences, and an invalid cookie being ignored.

## 2. Toolbar and result-action markup

- [x] 2.1 Merge GRID/LIST into one `VIEW:` link, add the `ACTIONS` toggle, mark sort and view links with `data-wga-aw-pref`, mark the card and row action wrappers with `wga-work-actions`, and render `<html data-aw-actions>` from the cookie. Verify with `templ generate` and focused templ render tests (`go test ./internal/assets/templ/... ./internal/handlers/artworks`).

## 3. Browser preference module and styles

- [x] 3.1 Add `registerSearchPrefs()` (storage, cookie, delegated listeners, toggle sync after HTMX swaps), call it from `app.ts`, and add the CSS that hides `.wga-work-actions` unless actions are on and keeps the toggle invisible until scripted. Verify with `bun test resources/js/search-prefs.test.ts`, `bunx biome check` on changed files, and `bun run build`.

## 4. Footer back to top

- [x] 4.1 Add `↑ BACK TO TOP` beside PREFERENCES, give the shared header `id="top"`, and remove the dead `.jump.back-to-top` handler and style after confirming nothing renders `.jump`. Verify with `templ generate`, a footer render test, and `bun run build`.

## 5. Browser verification

- [x] 5.1 Add or update Playwright specs for actions toggle on/off, the view switch, persistence across reload, no-JavaScript sort/view links, filter reset keeping preferences, and the footer link reaching `#top`; update specs that use search-result actions. Verify with `mise run test:playwright --port 8880` on the affected specs, then the full suite.

## 6. Preference-storage consent

- [x] 6.1 Add the optional `preferences` consent category and copy, gate every read and write of `wga_aw_prefs` and `wga-aw-prefs` on it, and delete both when consent is absent or withdrawn. Verify with `bun test resources/js/search-prefs.test.ts` and `mise run test:playwright --port 8880 playwright-tests/artwork-search-prefs-consent.spec.ts playwright-tests/cookieconsent.spec.ts`.

## 7. Final verification

- [x] 7.1 Run `go mod tidy`, `go vet ./...`, `go test ./... -cover`, and `golangci-lint run` with no new issues.
