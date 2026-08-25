# Visual-overhaul release acceptance inventory

## Rendering baseline

- Pixel-comparison browser: current stable Chrome desktop.
- Reference widths: 390px, 834px, and 1440px.
- Required themes: light and dark.
- Required conditions: default text, enlarged default font/text spacing, reduced motion,
  JavaScript disabled, and keyboard-only operation.
- Other supported-browser acceptance: current and previous Chrome, Edge, Firefox, macOS
  Safari, iOS Safari, and Android Chrome retain usable reference layout and behaviour.

## Route inventory

| Journey | Required route/surface | Acceptance evidence |
| --- | --- | --- |
| Enter collection | Home | Collection purpose, work of day, counts, recent additions, three discovery routes. |
| Browse artists | Artist index | Letter jump, filters, range, grid/table, unavailable rows, responsive state. |
| Examine artist | Artist record | Biography, terms, portrait, works/selections, citation, music card. |
| Read selection | Artist selection | Editorial preview, dedicated route, counts, commentary and citation. |
| Browse works | Artwork search | Filters, sort, grid/list, pagination, empty state and URL state. Tone-keyword exploration is deferred. |
| Examine work | Artwork record | Plate/viewer, provenance, metadata, image-derived palette, commentary, relation row, citation. Tone-keyword presentation is deferred. |
| Compare | Dual Mode | Complete panes, independent state, pane routing/history, image size, wide override. |
| Quick find | Search/palette | Artists and works split results, section and record keyboard lookup. |
| Explore dates | Timeline | Range, density, lanes, panels, record links and no-JavaScript submission. |
| Discover freely | Inspiration | Published set and clear route distinction from tours and itineraries. |
| Learn editorially | Guided-tour index and page | Filter/index, legacy state, title/text/picture/index/sources pages, contents and page turns. |
| Read statistics | Statistics | Charts, equivalent tables, captions, real-data figures and responsive order. |
| Find definitions | Glossary and in-prose terms | A-Z/search, focus-visible definitions and help tips. |
| Understand archive | About, contributors, reference pages | Information architecture, complete content and public error treatment. |
| Build itinerary | Tray and builder | Add from supported surfaces, fifteen-stop limit, arrangement, narration, reload recovery. |
| Share itinerary | Publish, public view and slideshow | Token/expiry, moderation, viewer, arrow/Escape and no-JavaScript reading. |
| Send postcard | Compose, confirmation and recipient page | Validation, abuse handling, delivery status, real recipient URL and music option. |
| Participate | Guestbook and project feedback link | Moderation, search/year browsing, and an ordinary floating link to the public GitHub issue list. |
| Listen | Period music player | No autoplay, single named player, fallback link and blocked-popup notice. |
| Set preferences | Footer/cookie surfaces | Theme, bionic reading, consent, persisted and unavailable-script states. |

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
   citation, palette, and four relationship bases.
2. Compare two complete records in Dual Mode, including a zoomed-text wide override,
   then share the pane state.
3. Read a selection and guided-tour source page, inspect statistics tables, and use
   glossary definitions without leaving the source prose.

## Accessibility acceptance

- Keyboard-only use in every supported desktop browser.
- NVDA with Firefox on Windows.
- VoiceOver with Safari on current macOS and iOS.
- TalkBack with Chrome on current Android.
- Dialog/viewer initial focus, tab containment, Escape/visible dismissal, background
  inertness, and focus restoration.
- No control or information is lost at enlarged text, 400% reflow, or reduced motion.
