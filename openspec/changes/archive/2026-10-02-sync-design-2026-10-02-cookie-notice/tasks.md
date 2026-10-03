# Tasks

## 1. Notice

- [x] 1.1 Keep the notice's `box inline` layout, which already places its text above its actions. In `resources/css/style.pcss`, replace the column-at-narrow rule and the 45rem row switch with wrapping button groups:
  - `ACCEPT ALL` is filled with the accent colour;
  - `DENY` and `PREFERENCES` are outlined at the same size;
  - every action is at least 44px tall.

  Verify with `bun run build` and a Playwright check in `playwright-tests/cookieconsent.spec.ts`:
  - at 390px: `ACCEPT ALL` and `DENY` share the first row at equal width, and `PREFERENCES` spans the row beneath;
  - at 1440px: all three share one row in order;
  - at both widths: every action is at least 44px tall, and only `ACCEPT ALL` has a filled background.

- [x] 1.2 Confirm that the notice keeps its bottom-left position and its bottom-stack offset. Re-measure the stack when the library reveals the notice, so it clears the permanent keyboard bar. Verify with `mise run test:playwright playwright-tests/bottom-stack.spec.ts`, including a test on the real revealed notice (30px bar plus 16px gap at 1440px, with no idle re-measurement).

## 2. Preferences interface

- [x] 2.1 Configure `preferencesModal` as `bar` with `position: right`. Style it in `style.pcss` to match the Preferences panel: full height at the right edge on desktop, full screen at or below 640px. Lay its actions out like the notice's. Verify with a Playwright check at 1440px (the panel's right edge meets the viewport's and it spans the full height) and at 390px (it covers the viewport).
- [x] 2.2 Give the close control a visible `CLOSE` label, and move focus to it when the interface opens, whether from the notice's `PREFERENCES` or from the footer's "Cookie settings". Verify with a Playwright check:
  - focus is on `CLOSE` after each way of opening it;
  - `Tab` cannot move focus out of the interface;
  - a click on the page behind it does nothing;
  - `Esc` closes it.
- [x] 2.3 Make closing the interface before any consent choice show the notice again. Verify with a Playwright check that opens it from the notice, closes it with `CLOSE` and then with `Esc`, and finds no consent recorded and the notice visible.

## 3. Preferences panel storage note

- [x] 3.1 In `footer.templ`, render both note variants: visit-only while storage is off, kept on this device while it is on. Make the consent decision handler set `data-wga-preference-storage="on|off"` on `<html>`, and add the CSS that shows the matching variant. Without the attribute, the visit-only note shows.

  Verify:
  - `templ generate` and `bun run build` complete;
  - the focused footer component test passes: `go test ./internal/assets/templ/components -run Footer`;
  - a Playwright check in `preference-storage-consent.spec.ts` sees the visit-only note before consent, and the on-device note after `ACCEPT ALL` on the same page.

## 4. Privacy policy

- [x] 4.1 Add a timestamped migration in `internal/migrations/` that replaces the privacy policy's `Cookies and Web Beacons` section with the new copy and leaves every other section untouched. When the section is missing, it logs that it skipped the record and makes no change. Its rollback returns an error, like `seed_about_page`.

  Verify with a focused `go test ./internal/migrations -run PrivacyCookie` covering three cases:
  - the section is replaced and the other sections are byte-identical;
  - the heading is absent, so nothing changes;
  - a re-run makes no further change.

- [x] 4.2 Check the rendered page. Verify with a Playwright assertion that `/pages/privacy-policy` names the session and consent record as essential, describes preferences as optional consent-gated storage, no longer has the "Cookies and Web Beacons" heading, and keeps its other sections.

## 5. Completion

- [x] 5.1 Update `docs/design-sync.md`: set "Last implemented" to the 2 October 2026 entry "Cookie notice: three-button consent, and preference storage needs it" with this change, and remove its in-progress row.
- [x] 5.2 Run the full verification:
  - `go mod tidy`;
  - `go vet ./...`;
  - `go test ./... -cover`;
  - `mise run check`;
  - `bunx prettier --check` on the changed Markdown;
  - `mise run test:playwright` for the full suite.
