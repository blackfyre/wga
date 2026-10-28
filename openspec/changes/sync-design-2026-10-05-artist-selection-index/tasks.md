# Tasks

## 1. Form grouping

- [x] 1.1 In `internal/handlers/artists`, add a pure function that takes ordered selection titles and work counts and returns forms. Add a named large-holding threshold constant of `10` beside it.
  - **Form order:** forms appear in first-appearance order, matched case-insensitively.
  - **Form contents:** each form holds its ordered selections, its selection count and its work total.
  - **Labels:** each selection's qualifier or full-title label is parsed as the spec states. Strip any trailing count, then try the final bracketed group, then the first colon, and otherwise use the full title. Hyphenated year ranges are shown with an en dash.

  Verify with a table test, `go test ./internal/handlers/artists -run '^TestGroupSelectionsByForm$'`, covering:
  - bracketed and colon qualifiers;
  - trailing counts after `·`, a dash and a comma, including thousands separators;
  - the en dash;
  - case-insensitive grouping and first-appearance order;
  - a title that does not parse;
  - `Woodcut series: The Revelation of St John (Apocalypse) (1497-98)`;
  - Dürer's four `Workshop in Nuremberg; paintings (…)` titles folding into one form.

- [x] 1.2 Branch the artist record's selection assembly on the threshold.
  - **Two to nine selections:** keep today's previews unchanged.
  - **Ten or more:** build the form-grouped view model. Each selection carries its route, tile anchor, label, work count and one cover image from its first work. Each form carries a stable anchor. Do not load the four-plate previews or sanitise commentary on this path.

  Verify with `go test ./internal/handlers/artists -run 'ArtistRecord|Selection'`. Add integration cases for nine selections (previews kept) and for ten selections with a shared form (grouped forms, counts and routes).

## 2. Artist record rendering

- [x] 2.1 In `internal/assets/templ/pages/artist.templ`, render large holdings as one block per form, in place of `CURATED SELECTIONS`.
  - Each block shows the form name, a `N SELECTIONS · N WORKS` header and a grid of square tile links to the selection routes.
  - Each tile link carries an `hx-get` that matches the existing record links.
  - Each tile shows its label and an `N works` / `1 work` count, and falls back to the form name when it has no cover.
  - Leave the preview markup for two to nine selections untouched.

  Run `templ generate`, then verify with `go test ./internal/assets/templ/pages -run 'ArtistRecord'`. Keep the existing preview tests and add form-block and tile tests, including the missing-cover fallback.

- [x] 2.2 For large holdings, render `ON THIS PAGE` as Biography, then one `FORM · N SEL · N` row per form, then Cite This Record.
  - A foldable form renders its selection entries open, in a two-column list, with no toggle.
  - A single-selection form has no list.
  - Keep the per-selection rows for previews, and the sticky rail and bottom-stack rules for both presentations.

  Verify with `go test ./internal/assets/templ/pages -run 'ArtistRecordTOC'`. Cover the form-row text, entry links to tile anchors, the open no-JS list, the unchanged preview rows, and the omission rules for zero and one selection.

## 3. Folding and scroll tracking

- [x] 3.1 Add a `resources/js` module, registered from `app.ts`, that acts only on a large-holding rail.
  - It adds a `+`/`−` button with `aria-expanded` to each foldable form row.
  - It collapses every form except the current one.
  - It tracks the current form with an IntersectionObserver on the form blocks, never on the tiles, highlighting the current row and opening it if it folds.
  - It is idempotent across `#mc-area` swaps, and disconnects its observer when the rail leaves the document.

  Verify with a `bun test` unit test of current-form selection and toggle state, then `bun run build`.

## 4. Browser verification

- [x] 4.1 Run `mise run test:playwright --port <port> playwright-tests/artist-record.spec.ts playwright-tests/selection.spec.ts playwright-tests/dual-mode.spec.ts`. This confirms that the curated fixture artist's three-selection previews and Dual Mode are unchanged.
  - If the synthetic fixture has gained an artist with ten or more selections, including a shared form, extend `artist-record.spec.ts` at 390px, 834px and 1440px. Cover the form blocks, tile links, fragment navigation, `+`/`−` and `aria-expanded`, scroll highlighting, the sticky rail clearing the bottom stack, and the fully listed no-JS index.
  - Otherwise, record in the change that the index has no browser coverage, and make no visual or interaction claim for it.
  - **Outcome:** the three specs pass (44 tests). The synthetic fixture's largest holding still has three published selections, so the large-holding index has no browser coverage. Go, templ and `bun test` unit tests cover it, and no visual or interaction claim is made for it.

## 5. Records and final checks

- [x] 5.1 Update `docs/design-sync.md` so that the 5 October 2026 entry "Artist page: large holdings get a selection index, grouped in the browser" is the last implemented one, with this change, and remove its in-progress row. Update any `docs/features/` text that describes artist-record selection previews. Run `bunx prettier --write` on the changed Markdown.
- [x] 5.2 Run `go mod tidy`, `go vet ./...`, `go test ./... -cover` and `mise run check`, and confirm that all pass.
