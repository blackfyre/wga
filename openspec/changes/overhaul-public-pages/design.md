## Context

The ignored hand-off bundle at `tmp/visual-overhaul/project/` is the acceptance reference. `WGA Prototype.dc.html` describes the full public experience; its exported `templ/` directory is a partial design aid, not a compatible implementation. In particular, its artwork search, Dual Mode, feedback, cookie, and glossary contracts do not match the current application.

The application is a server-rendered PocketBase/Templ monolith enhanced by HTMX and a small browser bundle. Public workflows already have route, DTO, query-string, partial-swap, and persistence contracts. The replacement must make the reference the visual and interaction contract without discarding richer behaviour which the prototype represents only superficially.

The user has decided that reference-defined functionality is binding. Cookie consent is the exception: it must gain the reference treatment while retaining the current Vanilla CookieConsent persistence and consent lifecycle. About remains static content. Glossary data already exists but lacks a public handler.

### Implementation baseline

| Public surface                                                      | Current owner and rendered contract                      | Replacement boundary                                                               |
| ------------------------------------------------------------------- | -------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `/`                                                                 | landing handler → `HomePageWrapped`                      | Home composition and shared shell                                                  |
| `/artists`, `/artists/{name}`, `/artists/{name}/{awid}`             | artists handler → artist list, artist page, artwork page | Catalogue record/list presentation; retain canonical URLs and glossary annotations |
| `/artworks`, `/artworks/results`                                    | artworks handler → search page/result block              | New search/filter/view contract and result block                                   |
| `/dual-mode`, dual lookup endpoint                                  | dual handler → pane composition, lookup, artwork block   | Visual replacement only; preserve pane URL state and operations                    |
| `/postcard`, `/postcard/send`                                       | postcard handler → compose/received dialog flow          | Dialog presentation only; preserve queue and delivery workflow                     |
| `/inspire`, `/statistics`, `/contributors`, `/open-source-licences` | dedicated handlers → page components                     | Public visual replacement; statistics IDs/data nodes remain stable                 |
| `/guestbook`, `/guestbook/add`, `/feedback`, `/pages/{slug}`        | guestbook, feedback, and static handlers → pages/dialogs | Public visual replacement plus new feedback and static-content capabilities        |
| Errors                                                              | shared error rendering → error page components           | Shared visual replacement                                                          |

The browser bundle initialises HTMX/dialog/toast/dual/glossary behaviour in `bootstrap.ts`, statistics in `statistics.ts`, and consent in `cookieconsent.ts`. Public partial updates currently use feature-owned containers such as `#mc-area`, `#artwork-search-results`, `#dual-area`, and `#d`; replacements must update the handler and browser contract together.

### Visual acceptance matrix

| Viewport            | Required review states                                                                                                                              |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Desktop, 1440 × 900 | Shared shell, every public route, full navigation, dialogs, artwork grid/list, Dual Mode, statistics, static contents navigation, and cookie notice |
| Mobile, 390 × 844   | Header/menu, page hierarchy, forms, dialogs, artwork grid/list, Dual Mode stacking, static contents navigation, and cookie notice                   |
| Reduced motion      | Shared shell, dialogs, feedback acknowledgement, and consent transitions at either viewport                                                         |
| JavaScript disabled | Public navigation, catalogue filter submission, glossary search submission, and static-page links                                                   |

## Goals / Non-Goals

**Goals:**

- Deliver the reference's Rams-inspired public interface consistently across every public page, responsive breakpoint, dialog, and error page.
- Add binding public capabilities for glossary browsing/search, redesigned catalogue exploration, and categorised contextual feedback.
- Keep public URLs, HTMX navigation, no-JavaScript links/forms, existing postcard delivery, and Dual Mode's complete state model operational unless the reference explicitly replaces them.
- Make visual and behavioural verification explicit through focused Go tests, semantic Playwright tests, and viewport-level review.

**Non-Goals:**

- Copy prototype React/inline-style code, bundle artefacts, placeholder data, or generated Templ files into the application.
- Replace PocketBase, HTMX, Templ, Chart.js, or Vanilla CookieConsent.
- Invent glossary metadata or artwork relationships absent from the stored collection.
- Change consent categories, introduce server-side consent routes, or make an external issue tracker the feedback system without a separate approved integration decision.
- Redesign administrative or internal PocketBase interfaces.

## Decisions

### Treat the prototype as an outcome specification, not source code

Implement the page geometry, tokens, copy, responsive states, and interactions from `WGA Prototype.dc.html` in the repository's Templ, Tailwind 4/daisyUI, HTMX, and TypeScript conventions. Use the supplied theme and Templ exports only as reference material, adapting them to current DTOs and route contracts.

This avoids importing incompatible DTOs and endpoints such as the prototype's simplified `/dual` workflow. Copying the exported files was rejected because they would replace live state contracts rather than express the design through them.

### Establish one public visual system at the layout boundary

Replace the global Templ layout, navigation, footer, dialog presentation, and shared controls first. Define the bone/ink/blue token set, square geometry, type hierarchy, hairlines, responsive shell, reduced-motion treatment, and shared interactive states in `resources/css/style.pcss`; public pages then consume semantic utility classes rather than page-specific hex values.

All public pages, partial response blocks, and error pages SHALL use the same shell vocabulary. Generated `*_templ.go` files remain derived output.

### Preserve progressive enhancement and existing HTMX transport contracts

Every enhanced public action retains an ordinary link or form submission target. Templ pages expose stable top-level swap blocks; HTMX selectors, targets, push URLs, and emitted toast/dialog events are updated as a coordinated contract with their handlers and browser initialisation.

Where the reference adds controls, handlers receive and validate the new inputs, create the required view DTO, and render either a whole page or the documented block. Request handlers remain framework adapters; non-trivial query and presentation transformation belongs in the owning feature package, which owns its persistence access and exposes explicit contracts to other features.

### Expand catalogue search while preserving Dual Mode state

The artwork feature owns the reference's query, taxonomic filters, date range, view choice, pagination, result presentation, and Dual Mode hand-off. The new controls will be translated onto an explicit feature-level filter model rather than reuse the prototype's incompatible DTO.

Dual Mode retains `/dual-mode`, its left/right paths, render targets, chooser, lookup, copy, reverse, clear, and pane-placement semantics. The reference's synchronisation concept is out of scope; the visual layout is layered onto the existing model rather than replacing it with the prototype's reduced `/dual` contract.

### Add Glossary as an owned read feature

Create a glossary handler package registered with the public handlers. It owns query parsing, collection reads, and the glossary page DTO; it does not route through the static-page handler. The initial read model uses the persisted `expression` and `definition` fields and derives alphabetical grouping from expressions.

The Glossary collection contains no fields beyond `expression` and `definition`. The prototype's category, use count, and “in the collection” link are therefore excluded rather than fabricated.

### Seed and structure reference static content

Map reference destinations without an owning data-backed feature to static-page records and add their content to the fresh-data seed path. The static-page handler SHALL process the content with the existing `withTableOfContents` helper before rendering so every `h2` and `h3` has a stable fragment identifier and the page can render a matching hierarchical contents navigation.

### Use the reference feedback interaction with the existing feedback workflow

Render a contextual feedback dialog with category selection, visible character budget, optional contact details, and an acknowledgement state. Map its fields through the feedback feature's request input and persist the report category with the report. Preserve honeypot validation, request-origin context, toast handling, and error recovery.

An existing untracked migration, `1784808003_feedback_categories.go`, introduces the expected category values and optional contact fields. Implementation must inspect and either adopt or replace it deliberately; it must not duplicate or silently overwrite that user-owned work.

The existing migration is adopted unchanged: its required `general`, `correction`, `technical`, and `suggestion` categories and optional name/email fields match the selected report model.

The reference's statement that reports go to a public issue tracker is presentation copy, not an approved external side effect. Persisting reports in the existing feedback workflow is the selected scope until an issue-tracker integration is separately specified.

### Skin Vanilla CookieConsent instead of replacing it

Keep `vanilla-cookieconsent` as the sole consent controller. Configure its supported modal position and labels, and override its documented CSS variables and structural classes to render the reference panel. Retain the present read-only `necessary` category, its cookie storage, and preference modal.

The prototype's separate “accept all” and “essential only” semantics cannot be reproduced honestly with a single necessary category. The implementation uses the reference panel geometry and button styling with the library's current meaningful actions rather than introducing false analytics choices or nonexistent `/cookies/*` handlers.

### Test behaviour independently from markup and visual acceptance

Replace CSS-structure-coupled Playwright locators with roles, labels, values, URLs, and outcome assertions. Retain tests for postcard sending/delivery, guestbook and feedback submission, cookie consent, theme/reduced-motion behaviour, statistics, catalogue search, and every Dual Mode operation; add glossary and new catalogue/filter coverage.

Add page-level browser assertions at desktop and mobile widths for public shell visibility and responsive navigation. Visual parity is accepted against the hand-off source through deliberate viewport review, because the bundle provides no committed screenshot baseline.

## Risks / Trade-offs

- [The reference is complete visually but its exported code has stale contracts] → Use the prototype for outcomes and map each interaction to the live feature contract before replacing a page.
- [A broad shell replacement can break partial swaps or browser initialisation] → Establish shared block and lifecycle conventions first, then migrate routes in groups with focused browser checks.
- [Dual Mode carries unusually high URL-state complexity] → Preserve its current handler tests and full Playwright scenario suite while changing only its presentation and explicitly approved controls.
- [Feedback category migration already exists outside version control] → Review it before implementation; stage only the final intentional migration and test a fresh data directory.
- [The existing Glossary schema has only expression and definition] → Render only those stored facts and omit unsupported category, usage, and artwork-link treatments.
- [Reference static destinations are not seeded] → Add the required static-page records to the fresh-data seed path and verify a newly initialised data directory renders them with a generated contents navigation.
- [Cookie visual parity conflicts with its single consent category] → Retain truthful Vanilla CookieConsent semantics and document the deliberate button-copy difference.
- [A large template rewrite can weaken accessibility] → Keep semantic landmarks, visible labels, focusable controls, dialog lifecycle, reduced-motion support, and no-JavaScript routes under browser test.

## Migration Plan

1. Build and test the new CSS/JS/Templ assets alongside focused handler tests before deployment.
2. Apply any intentional feedback schema migration through the normal PocketBase migration command; test a fresh data directory and an existing data directory upgrade.
3. Deploy the single application artefact. Existing public URLs continue to resolve; client assets are embedded with the binary.
4. Smoke-test the public shell, consent modal, glossary, catalogue search, Dual Mode, postcards, feedback, and guestbook in the deployed environment.
5. Roll back by redeploying the prior application artefact. Do not roll back a feedback migration automatically when it contains preserved user reports; use a forward-compatible corrective migration if needed.
