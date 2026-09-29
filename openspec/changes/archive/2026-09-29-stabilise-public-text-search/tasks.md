# Tasks

## 1. Shared building blocks

- [x] 1.1 Add a helper in `internal/utils` that reports whether a request's `HX-Trigger` header equals a given element id. Cover matching, a different id, a missing header and whitespace in `internal/utils/main_test.go`. Verify with `go test ./internal/utils -run '^TestRequestTriggeredBy'`.
- [x] 1.2 Add a preserve flag to `dto.Field` and render `hx-preserve` from `components.Field` when it is set, leaving the attribute absent otherwise. Add a component test for both cases. Verify with `templ generate` and `go test ./internal/assets/templ/components`.

## 2. Artworks search

- [x] 2.1 Change the `#artwork-filters` trigger to `change, input changed delay:500ms`. Give the collection query field a stable id. Render `hx-preserve` on `q`, `technique` and the collection query only when the view requests it. Update the trigger assertion in `internal/assets/templ/pages/artworks_test.go` and add cases with and without preservation. Verify with `templ generate` and `go test ./internal/assets/templ/pages -run 'Artwork'`.
- [x] 2.2 In the artworks handler, set preservation when `HX-Trigger` is `artwork-filters`, in both standalone and Dual Mode rendering. Add handler tests showing that a form-issued request preserves the fields and that a request without that trigger does not. Verify with `go test ./internal/handlers/artworks`.

## 3. Artists index

- [x] 3.1 Change the `#artist-filters` trigger to `delay:500ms` for search input (the range trigger is unchanged) and add `hx-sync="this:replace"`. In the artists handler, set the name field's preserve flag when `HX-Trigger` is `artist-filters`. Add handler tests covering form-issued and letter/sort/reset requests. Verify with `templ generate` and `go test ./internal/handlers/artists`.

## 4. Dual Mode artist index

- [x] 4.1 Change the `dual-filters-<key>` trigger to `delay:500ms` for search input (the range trigger is unchanged) and add `hx-sync="this:replace"`. Set the pane's name-field preserve flag only when `HX-Trigger` equals that pane's form id, so the other pane and BACK/INDEX/RESET requests render plainly. Add handler tests for each pane and for a non-form request. Verify with `templ generate` and `go test ./internal/handlers/dual`.

## 5. Guestbook and glossary search

- [x] 5.1 Give the guestbook search form and its `q` field stable ids. Change the trigger to `delay:500ms`, add `hx-sync="this:replace"`, and render `hx-preserve` on `q` only when `HX-Trigger` is the search form's id. The entry form's requests must not preserve it. Add handler tests. Verify with `templ generate` and `go test ./internal/handlers/guestbook`.
- [x] 5.2 Give the glossary form and its `q` field stable ids. Change the trigger to `input changed delay:500ms`, add `hx-sync="this:replace"`, and render `hx-preserve` on `q` only when `HX-Trigger` is the glossary form's id, so A–Z links render plainly. Add handler tests. Verify with `templ generate` and `go test ./internal/handlers/glossary`.
- [x] 5.3 Change the global search trigger (`search.templ`) to `input changed delay:500ms`, with no preservation change, and update any assertion on the old trigger. Verify with `templ generate` and `go test ./internal/assets/templ/pages ./internal/handlers/search`.

## 5A. Text-search script and blur filter

- [x] 5A.1 Add a `TextSearch` flag to `dto.Field`, rendered by `components.Field` as `data-text-search`. Render the marker on every affected free-text field: artworks `q`, `technique` and collection query; the artists and Dual Mode name fields; the guestbook and glossary `q`. Add the filter `[!target.hasAttribute('data-text-search')]` to each affected form's `change` trigger, and update trigger and marker assertions in the Go template and component tests. Verify with `templ generate` and `go test ./internal/assets/templ/... ./internal/handlers/...`.
- [x] 5A.2 Add `resources/js/text-search.ts`, an idempotent, document-level initialiser called once from `bootstrap.ts`. It records each requesting form's field values on `htmx:beforeRequest`. When that form's own response settles, it sets the fields' `value` attributes to the recorded values, and re-dispatches `input` for any field whose live value differs. Verify with `bunx biome check resources/js/text-search.ts resources/js/bootstrap.ts` and `bun run build`.

## 6. Browser verification

- [x] 6.1 Add Playwright coverage to `artwork-search.spec.ts`, `artists.spec.ts`, `dual-mode.spec.ts`, `guestbook.spec.ts` and `glossary.spec.ts`:
  - type a word with `pressSequentially` using a per-key delay shorter than 500 ms;
  - assert that one search request is issued after the pause;
  - assert that the field keeps the full value, focus and caret after the swap;
  - assert that typing during an in-flight request is not lost;
  - assert that reset or navigation clears or replaces the field;
  - assert that browser back restores the earlier value.

  Verify with `mise run test:playwright -- artwork-search artists dual-mode guestbook glossary`.

## 7. Final verification

- [x] 7.1 Check `docs/features/` and `docs/development-guide.md` against `docs/documentation-maintenance.md` and update any description of search debounce or trigger behaviour. Then run `go mod tidy`, `go vet ./...`, `go test ./... -cover` and `mise run check`, all with passing results.
