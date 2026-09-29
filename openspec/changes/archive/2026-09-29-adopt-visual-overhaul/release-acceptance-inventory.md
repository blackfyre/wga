# Visual-overhaul release acceptance inventory

## Rendering baseline

- Pixel-comparison browser: current stable Chrome desktop.
- Reference widths: 390px, 834px, and 1440px.
- Required themes: light and dark.
- Required conditions: default text, enlarged default font/text spacing, reduced motion,
  JavaScript disabled, and keyboard-only operation.
- Machine-verifiable browser acceptance: current stable Chrome and Playwright-managed
  Firefox on Linux retain usable reference layout and behaviour; Chromium Android-device
  emulation covers the mobile layout and touch contract.

## Route inventory

| Journey            | Required route/surface                   | Acceptance evidence                                                                                                                                               |
| ------------------ | ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Enter collection   | Home                                     | Collection purpose, work of day, counts, recent additions, three discovery routes.                                                                                |
| Browse artists     | Artist index                             | Letter jump, native alphabetical school/period selects, range, grid/table, unavailable rows, responsive state, and public-field focus.                            |
| Examine artist     | Artist record                            | Biography, terms, portrait, works/selections, citation, music card.                                                                                               |
| Read selection     | Artist selection                         | Editorial preview, dedicated route, counts, commentary and citation.                                                                                              |
| Browse works       | Artwork search                           | Filters, sort, grid/list, pagination, empty state and URL state. Tone-keyword exploration is deferred.                                                            |
| Examine work       | Artwork record                           | Plate/viewer, provenance, metadata, image-derived palette, commentary, relation row, citation. Tone-keyword presentation is deferred.                             |
| Compare            | Dual Mode                                | Complete records and selections, namespaced pane contents, independent state, pane routing/history, location/zoom/citation notes, and wide override.              |
| Quick find         | Search/palette                           | Artists and works split results, section and record keyboard lookup.                                                                                              |
| Explore dates      | Timeline                                 | Range, density, lanes, panels, record links and no-JavaScript submission.                                                                                         |
| Discover freely    | Inspiration                              | Published set and clear route distinction from tours and itineraries.                                                                                             |
| Learn editorially  | Guided-tour index and page               | Filter/index, legacy state, title/text/picture/index/sources pages, contents and page turns.                                                                      |
| Read statistics    | Statistics                               | Charts, equivalent tables, captions, real-data figures and responsive order.                                                                                      |
| Find definitions   | Glossary and in-prose terms              | A-Z/search, focus-visible definitions and help tips.                                                                                                              |
| Understand archive | About, contributors, reference pages     | Information architecture, complete content and public error treatment.                                                                                            |
| Build itinerary    | Tray and builder                         | Add from supported surfaces, fifteen-stop limit, arrangement, narration, reload recovery.                                                                         |
| Share itinerary    | Publish, public view and slideshow       | Token/expiry, moderation, viewer, arrow/Escape and no-JavaScript reading.                                                                                         |
| Send postcard      | Compose, confirmation and recipient page | One-to-five-recipient composition, validation, abuse handling, independently retryable delivery status, masked addresses, and one protected shared recipient URL. |
| Participate        | Guestbook and project feedback link      | Moderation, search/year browsing, and an ordinary floating link to the public GitHub issue list.                                                                  |
| Listen             | Period music player                      | No autoplay, single named player, fallback link and blocked-popup notice.                                                                                         |
| Set preferences    | Footer/cookie surfaces                   | Theme, bionic reading, consent, persisted and unavailable-script states.                                                                                          |

## Persona journeys

### Regular visitor

1. Open Home, discover a work through Inspiration or search, inspect an artwork, and
   return through an artist or related-work link.
2. Add works from browsing and record surfaces, arrange/narrate a fifteen-stop
   itinerary, publish it, and open it as a recipient.
3. Send a postcard, sign the guestbook, and submit contextual feedback without an
   account.

### Scholar

1. Use artwork filters and sort state; share the URL; inspect record provenance,
   citation, palette, and three relationship bases.
2. Compare two complete records in Dual Mode, including a zoomed-text wide override,
   then share the pane state.
3. Read a selection and guided-tour source page, inspect statistics tables, and use
   glossary definitions without leaving the source prose.

## Accessibility acceptance

- Keyboard-only use in stable Chrome and Playwright-managed Firefox.
- Semantic names, roles, states, and relationships asserted through browser accessibility
  locators and DOM contracts.
- Dialog/viewer initial focus, tab containment, Escape/visible dismissal, background
  inertness, and focus restoration.
- No control or information is lost at enlarged text, 400% reflow, or reduced motion.
- No-JavaScript fallbacks retain navigation, content, and state-changing links.

Physical-device, Edge, Safari, NVDA, VoiceOver, and TalkBack certification is outside
this change and may be performed as non-blocking downstream assurance.

## Clean `016cf6f` affected-surface refresh

The final affected comparison pins a clean reference worktree at
`016cf6f0e93e88ce173bff3f9e8a09d854e52d35`. Chrome 152 captured the artist
index, Dual Mode, and postcard composer in both light and dark schemes at 390px,
834px, and 1440px: 36 application/reference screenshots in total. Inspection
confirmed the reference composition, control hierarchy, responsive geometry, and
palette treatment without horizontal overflow. Synthetic fixture content and
unavailable staged fixture media were treated as data differences rather than
template drift.

| Affected surface | Refreshed acceptance evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Artist index     | Shared native SCHOOL and PERIOD selects retain alphabetical/type-ahead selection, visible palette-aware focus, HTMX URL state, ordinary JavaScript-disabled GET submission, dark native-menu treatment, and overflow-free 390px/834px/1440px composition.                                                                                                                                                                                                                                                                   |
| Dual Mode        | Both panes retain independent index, routing, history, share/reload, and wide state. Complete artist/artwork records, curated selection groups and pane-local selection pages expose namespaced Biography/selection/citation contents, sibling navigation, current location, zoom guidance, original download, and citation access notes. Retired size parameters canonicalise away and no image-size control or preference copy remains.                                                                                   |
| Postcard         | One to five recipient rows work through HTMX and the same ordinary form endpoint; keyboard and JavaScript-disabled add/remove/submission paths retain values on correction. Successful Mailpit delivery creates independently retryable per-address work, masks every confirmation address, and gives both recipients the same protected URL. CAPTCHA rejection remains actionable, arbitrary recipient tokens fail closed, sender-controlled music is absent, and recipient music derives only from the published artwork. |

### Final focused verification

- Templ generation and the production frontend build passed with no generated-source
  updates required.
- 746 focused Go tests passed across postcard workflow/handlers, migrations,
  repositories, artist handlers, Dual Mode handlers, and affected Templ packages;
  `go vet ./...` passed.
- Biome passed the three affected Playwright files. Strict OpenSpec validation and
  scoped diff checks passed.
- The affected Chromium run passed 34/34 scenarios, split only by required server
  configuration: 32 artist, Dual Mode, postcard composition/delivery, responsive,
  keyboard, HTMX, JavaScript-disabled, dark-palette, shared-link, and retired-URL
  scenarios with CAPTCHA disabled; two CAPTCHA correction/rejection scenarios with
  Google's test configuration. The separate production-shaped Dual Mode suite passed
  10/10 scenarios.
