## Why

The public site currently presents an inconsistent, component-library-led interface that does not match the supplied Rams-inspired visual reference. The reference defines a coherent public experience and introduces useful catalogue, glossary, and feedback interactions that should become the product rather than remain a prototype.

## What Changes

- Replace the public Templ shell, pages, components, CSS theme, and browser interactions with the supplied visual reference as the visual and interaction contract. Use the updated reusable-component bundle as a pattern reference, adapting or creating local primitives only where they preserve WGA's existing DTO, route, HTMX, and accessibility contracts.
- Rebuild responsive navigation, home, artist and artwork browsing, artwork records, postcards, inspiration, statistics, static content, guestbook, contributors, feedback, errors, and dialogs in the new visual language.
- Add the data-backed Glossary route with A–Z filtering and delayed text search over the existing Glossary collection.
- Seed the reference's missing static-content destinations and render their generated table of contents in the redesigned static-page experience.
- Replace artwork search with the reference's live filters, year range, grid/list views, reset behaviour, pagination, and Dual Mode hand-off while retaining the richer existing Dual Mode URL and pane-placement model.
- Extend feedback to support the reference's report category, source context, character count, and acknowledgement experience.
- Restyle Vanilla CookieConsent to the reference's cookie-notice treatment while retaining its current client-side persistence and consent behaviour.
- Update browser helpers, Templ DTOs and handlers only where the reference requires new interaction or data contracts.
- Rewrite Playwright coverage around the new semantic UI, preserving existing end-to-end workflows and adding coverage for new public interactions.

## Capabilities

### New Capabilities

- `public-page-experience`: The coherent public visual system, responsive shell, page composition, dialogs, static content presentation, and accessible interaction treatment for all public routes.
- `catalogue-exploration`: The redesigned artwork search and comparison experience, including the reference's new search controls while preserving the existing Dual Mode state model.
- `glossary-browsing`: A public, data-backed glossary with alphabetical browsing and live term/definition search.
- `public-feedback`: Categorised public feedback reports with contextual metadata and an accessible acknowledgement flow.

### Modified Capabilities

- None.

## Impact

- Templ sources under `internal/assets/templ/`, including layouts, pages, components, and error pages; generated `*_templ.go` files remain generated and uncommitted.
- Frontend sources in `resources/css/` and `resources/js/`, including the Vanilla CookieConsent configuration and current global `wga` helpers.
- Handler registration and the artwork, dual, feedback, glossary, static, landing, and related public handler packages.
- Existing public data and routes, particularly the `Glossary` collection, static-page seed data and table-of-contents rendering, feedback records, artwork search URLs, and Dual Mode URLs.
- All public Playwright specifications and focused Go handler tests.
