# Tasks

## 1. Shared preference reader

- [x] 1.1 Add `internal/visitorprefs` with the consent check, cookie presence check, and validated remembered view; move artwork search consent/cookie decoding onto it and verify `go test ./internal/visitorprefs ./internal/handlers/artworks` passes.

## 2. Server-side view resolution

- [x] 2.1 Artwork search: apply the remembered view to every page or `/artworks/results` request without a `view` parameter, emit `view=grid` when the fallback is list, and keep sort/direction bare-visit only; verify with `go test ./internal/handlers/artworks -run 'Prefs|View'`.
- [x] 2.2 Artist index: resolve the view from the parameter or fallback, emit grid explicitly when needed (links and the filter form's hidden view input), add `Vary: Cookie` and the private response when the cookie is present; verify with `go test ./internal/handlers/artists`.
- [x] 2.3 Dual Mode: default panes to grid, resolve `l_view`/`r_view` from the parameter or fallback, canonical URLs omit grid unless the fallback is list, reset keeps the fallback, add `Vary: Cookie` and the private response; verify with `go test ./internal/handlers/dual`.

## 3. Client recording and copy

- [x] 3.1 Add `data-wga-view-pref` to the artist index and Dual Mode toggles (`templ generate`) and record it in `resources/js/search-prefs.ts`, preserving other fields and the consent gate; verify with `bun test resources/js/search-prefs.test.ts` and `bun run build`.
- [x] 3.2 Update the cookie preferences copy in `resources/js/cookieconsent.ts` and add an idempotent migration rewording the privacy policy sentence; verify with `go test ./internal/migrations`.

## 4. Browser verification

- [x] 4.1 Add a Playwright journey: with consent, choosing list on `/artworks` makes `/artists` and a Dual Mode index pane render as lists; without consent both render as grids. Update affected specs and run `mise run test:playwright --port 8093` on them.
- [x] 4.2 Run `go vet ./...`, `go test ./... -cover`, `mise run check`, `bun test`, and `openspec validate universal-listing-view-preference`.
