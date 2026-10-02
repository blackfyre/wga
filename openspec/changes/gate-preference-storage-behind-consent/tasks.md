## 1. Shared consent gate

- [x] 1.1 Add `resources/js/preference-consent.ts` with the consent-record parser, gated storage helpers, cookie expiry, store registration and the grant/withdraw transition. Move `search-prefs.ts` onto it and point `cookieconsent.ts` at `applyPreferenceConsent`. Verify with `bun test resources/js/preference-consent.test.ts resources/js/search-prefs.test.ts`.

## 2. Gated stores

- [x] 2.1 Gate `wga-theme` and `wga-palette` in `appearance.ts`, register its store, and expire the legacy `wga_theme` and `wga_palette` cookies. Verify with `bun test resources/js/appearance.test.ts`.
- [x] 2.2 Gate `wga-bionic` in `bionic.ts`, register its store, and expire the legacy `wga_bionic` cookie. Verify with `bun test resources/js/bionic.test.ts`.
- [x] 2.3 Gate `wga-study-board` in `study-board.ts` and register its store. Stop a `?board=` URL from overwriting the remembered board, and restore the in-memory board on a bare `/study-board` without consent. Verify with `bun test resources/js/study-board.test.ts`.

## 3. Head resolver

- [x] 3.1 Make the inline resolver in `internal/assets/templ/dto/preferences.go` read `wga-palette` and `wga-theme` only while `cc_cookie` accepts `preferences`. Verify with `go test ./internal/assets/templ/dto ./internal/assets/templ/layouts`.

## 4. Copy

- [x] 4.1 Update the consent notice and the preferences dialog in `cookieconsent.ts`, the appearance panel note in `components/footer.templ`, and the Study Board lede in `pages/study_board.templ`. Verify with `templ generate`, `bun run build` and `go test ./internal/assets/templ/...`.

## 5. Browser verification

- [x] 5.1 Add Playwright coverage for each store. It covers no consent (nothing persists after reload), a grant (the rendered state is stored), a withdrawal (the copies are cleared), a grant or withdrawal in another tab, and a `?board=` URL that leaves a remembered board alone. Update the existing specs that relied on remembered state to grant consent through the shared helper. Verify with `mise run test:playwright --port 8895` on the affected specs, then the full suite.

## 6. Final verification

- [x] 6.1 Run `go vet ./...`, `go test ./... -cover`, `golangci-lint run`, `bunx biome check` on the changed TypeScript, and `bunx prettier --check` on the changed JS and Markdown.
