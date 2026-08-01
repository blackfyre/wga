## 1. Establish the migration baseline

- [x] 1.1 Inventory every public route, Templ page/block, dialog, form, HTMX target, browser initialiser, and Playwright journey against `WGA Prototype.dc.html` and record the final route-to-reference mapping.
- [x] 1.2 Review the existing untracked feedback category migration and related handler tests; decide whether to adopt it unchanged or replace it with one intentional migration before editing feedback persistence.
- [x] 1.3 Use the confirmed `expression` and `definition` Glossary schema as the complete initial read model and exclude unsupported prototype metadata.
- [x] 1.4 Define the public-page desktop and mobile viewport acceptance matrix, including the reference states that require manual visual review.

## 2. Build the shared public visual system

- [x] 2.1 Replace the active daisyUI/Tailwind theme and shared CSS with the reference tokens, square controls, typography hierarchy, hairlines, responsive rules, and reduced-motion treatment.
- [x] 2.2 Rebuild the Templ root layout, staging/build treatment, branded header, desktop navigation, mobile navigation, search affordance, footer, toast, and dialog presentation to match the reference.
- [x] 2.3 Update browser bootstrap and shared `wga` namespace types so global layout, dialog, HTMX lifecycle, and newly required UI helpers initialise after full and partial renders.
- [x] 2.4 Add focused browser coverage for desktop/mobile navigation, keyboard access, reduced motion, and JavaScript-disabled public navigation.

## 3. Migrate general public pages and components

- [x] 3.1 Rebuild the home page from live content and routes in the reference composition, including hero, browse/compare calls to action, featured artwork, collection figures, and public links.
- [x] 3.2 Rebuild artist listing, artist record, artwork record, and inspiration page presentation while retaining their existing links, image handling, metadata, and HTMX partial behaviour.
- [x] 3.3 Rebuild postcard compose and received-postcard views, preserving modal lifecycle, validation, queueing, delivery, and email pickup behaviour.
- [x] 3.4 Rebuild guestbook, contributor, licence, and static-page/error-page presentation; map reference destinations without a data-backed feature to static pages and retain each existing public interaction.
- [x] 3.5 Add the missing reference static-page records to the fresh-data seed path and verify every static navigation destination resolves after initialisation.
- [x] 3.6 Wire `internal/handlers/static/toc.go` into static-page rendering and rebuild the static-page Templ presentation for generated hierarchical contents navigation and stable heading fragments.
- [x] 3.7 Update semantic Playwright locators and assertions for migrated public routes without weakening their existing workflow coverage.

## 4. Deliver the redesigned catalogue exploration feature

- [x] 4.1 Define and test the artwork feature's explicit filter/query model for text, school, form, type, date range, selected view, paging, and Dual Mode context.
- [x] 4.2 Update artwork search handlers and DTOs to parse, validate, preserve, and render the redesigned filter state for full-page and HTMX requests.
- [x] 4.3 Rebuild artwork search and result Templ blocks with the reference filter rail, grid/list controls, result summary, empty state, reset, pagination, and no-JavaScript submit path.
- [x] 4.4 Preserve and test shareable URLs, clear/reset behaviour, result paging, and selected presentation after enhanced and non-enhanced searches.
- [x] 4.5 Extend Playwright catalogue tests for every new filter, date range, grid/list state, empty state, pagination, and Dual Mode hand-off.

## 5. Rebuild Dual Mode without regressing its state model

- [x] 5.1 Map the reference Dual Mode layout onto the existing `/dual-mode` left/right path and render-target model, excluding the reference's synchronisation concept.
- [x] 5.2 Rebuild Dual Mode page, panes, chooser, lookup results, target controls, operations, and manual-load forms in the reference visual system.
- [x] 5.3 Preserve handler contracts for pane replacement, same/other-pane targeting, copy, reverse, clear, and URL push state.
- [x] 5.4 Update focused Dual Mode handler tests and the full semantic Playwright comparison suite for the redesigned controls and all URL-state transitions.

## 6. Add Glossary browsing

- [x] 6.1 Create the Glossary feature's owned read model and query helpers for expression/definition retrieval, A–Z filtering, text search, ordering, and empty results.
- [x] 6.2 Register the public Glossary handler and render full-page or HTMX block responses with canonical/push URL metadata.
- [x] 6.3 Create the Glossary Templ page and reusable term presentation in the reference visual system, using only persisted glossary facts.
- [x] 6.4 Add focused Go tests and Playwright coverage for initial load, alphabetical filtering, delayed search, reset, empty results, semantic labels, and no-JavaScript form fallback.

## 7. Deliver feedback and cookie-consent treatments

- [x] 7.1 Apply the deliberate feedback category/contact schema decision and update the feedback workflow input, validation, persistence, context capture, and error handling.
- [x] 7.2 Rebuild the feedback trigger, dialog, category controls, message guidance, character count, honeypot fields, acknowledgement, and recovery states to match the reference.
- [x] 7.3 Add the browser helper functions and type declarations for feedback guidance and other reference interactions, preserving existing dialog and toast events.
- [x] 7.4 Restyle and configure Vanilla CookieConsent with its supported layout/CSS mechanisms while retaining the current necessary-consent and preferences semantics.
- [x] 7.5 Add focused Go tests and Playwright coverage for feedback categories, persistence recovery, cookie consent persistence, preferences, and truthful consent actions.

## 8. Complete statistics and accessibility-sensitive presentation

- [x] 8.1 Replace the statistics page with the reference-compatible presentation while preserving existing DTO fields, canvas IDs, JSON data nodes, Chart.js initialisation, and no-JavaScript data tables.
- [x] 8.2 Update statistics chart styling and browser behaviour to the shared visual system without changing published-data calculations.
- [x] 8.3 Verify every redesigned form has visible labels, every dialog has an accessible name and dismissal path, and all interactive states remain keyboard-operable.

## 9. Validate and release the overhaul

- [x] 9.1 Run `templ generate`, `bun run build`, focused Go tests, and `go test ./... -cover`; fix only failures attributable to this change.
- [x] 9.2 Run the public Playwright suite with the application, Mailpit, and required environment variables available; update assertions only to reflect approved reference behaviour.
- [x] 9.3 Perform desktop and mobile viewport review against `WGA Prototype.dc.html` for every public route and record any intentional deviations caused by retained live behaviour or unavailable data.
- [x] 9.4 Run `go vet ./...`, `golangci-lint`, and the complete build path; confirm generated Templ and public assets are not committed.
- [x] 9.5 Review the final diff for unrelated user changes, migration safety, accessibility regressions, stale prototype copy, and preserved public route contracts.
