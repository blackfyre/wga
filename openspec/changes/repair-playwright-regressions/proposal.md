# Proposal

## Why

Once the fixture-bound specs are repaired, the remaining Playwright failures fall into four groups (local run 2026-09-30; the same set fails in CI):

- **Admission budget exhaustion (6 tests).** Itinerary draft creation is limited to three per hour per client. The whole suite runs from one client, so later itinerary, study-board, and shell tests hit the "created too many itineraries" toast and the tray never appears.
- **Markup drift and one regression from the visual overhaul (9 tests).** The artwork record plate no longer carries `data-viewer-no-navbar`. `bootstrap.ts` still reads that attribute, so the viewer's thumbnail navigation bar has come back (5 tests). Other specs still assert behaviour the overhaul changed on purpose: file weight in FILE facts (no longer shown, per the catalogue-exploration spec), the "WGA" mark as the first logo span, and the footer's large-viewport column count.
- **No-JavaScript form submits never become "stable" (4 tests).** Guestbook, artwork search, timeline, and reference-page actions time out in Playwright's stability check even though the controls render.
- **Layout and interaction bugs (3 tests).** The artist record overflows at 200% text (512 px scroll width against a 390 px viewport), the cookie notice overflows at 390 px (395 px), and a keyboard-navigation test never observes its expected `/artworks/results` request.

The postcard spec's local failure (no Mailpit on `:8025`) is environmental and out of scope.

## What Changes

- Itinerary draft and publication admission budgets become deployment settings loaded through `internal/config`, with the current production values (3 per hour each) as defaults. The Playwright runner (`mise run test:playwright`) and the CI Playwright job raise them so one client can run the suite.
- The artwork record plate carries `data-viewer-no-navbar` again, so its viewer hides the thumbnail navigation bar.
- Specs that encode pre-overhaul markup are updated to the current reference: no file weight, the current logo mark structure, and the reference footer column count.
- The shared cause of the no-JavaScript "not stable" failures is found and fixed at its source.
- The artist-record 200%-text overflow and the 390 px cookie-notice overflow are fixed in the layout.
- The keyboard-navigation traversal test's expected request is made to match the current search request contract; if the application behaviour itself regressed, the application is fixed instead.

## Non-Goals

- Changing production admission limits.
- The postcard Mailpit dependency.
- Tests covered by `bind-playwright-specs-to-synthetic-fixture`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `visitor-itineraries`: admission budgets for drafts and publications become configurable, with unchanged production defaults.

## Impact

- `internal/config`, `internal/itineraries/admission.go`, and the itinerary handler wiring; `.env.example`; `mise.toml` (`test:playwright`); `.github/workflows/playwright.yml`; `docs/development-guide.md` configuration section.
- `internal/assets/templ/pages/artwork.templ`; the layout sources behind the artist record and cookie notice (`.templ` and `resources/css/style.pcss`).
- `playwright-tests/{artwork,public-navigation,public-shell,guestbook,artwork-search,timeline-task81,reference-pages,keyboard-navigation,artist-record,cookieconsent}.spec.ts`, as each fix requires.
