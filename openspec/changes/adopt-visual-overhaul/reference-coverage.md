# Visual-overhaul reference coverage audit

## Purpose

This audit records the production owner and current delivery status of every
non-development visual-overhaul surface before implementation. The reference's
`WGA Prototype.dc.html` is the intended public outcome; the paths below identify
the production extension point rather than a claim that the current page meets
that outcome.

## Public screens

| Reference surface | Production owner | Status |
| --- | --- | --- |
| Home | `internal/handlers/landing/main.go`; `internal/assets/templ/pages/home.templ` | Existing route; redesign and reference data required. |
| Artist index and record | `internal/handlers/artists/`; `pages/artists.templ`; `pages/artist.templ` | Existing routes; filtering, views, record composition, and citations require expansion. |
| Artist selections | — | No production handler, collection, route, or page. |
| Artwork search and record | `internal/handlers/artworks/`; `pages/artworks.templ`; `pages/artwork.templ` | Existing routes; reference filters, sort, record composition, relations, and plate behaviour require expansion. |
| Postcard compose and received | `internal/handlers/postcards/`; `internal/postcards/`; `pages/postcard.templ` | Existing workflow and page; reference's distinct compose, confirmation, and recipient surfaces require reconciliation. |
| Reference/static pages | `internal/handlers/static/`; `pages/static.templ` | Existing route type; About and reference information architecture require redesign and content audit. |
| Inspiration | `internal/handlers/landing/`; `pages/inspire.templ` | Existing semantic route; reference composition required. |
| Search results | `internal/handlers/search/`; `pages/search.templ` | Existing route; reference split result presentation required. |
| Dual Mode | `internal/handlers/dual/`; `pages/dual.templ` | Existing route/state model; complete-record panes, per-pane indexes and sizing require expansion. |
| Guestbook | `internal/handlers/guestbook/`; `pages/guestbook.templ`; `internal/hooks/guestbook.go` | Existing route and data; moderation, archive navigation, and visual contract require expansion. |
| Contributors | `internal/contributors/`; `pages/contributors.templ` | Existing page; reference content model and layout require expansion. |
| Artwork search | `internal/handlers/artworks/`; `pages/artworks.templ` | Existing route; filters, sort, and dense-list contract require expansion. Tone-keyword exploration is deferred. |
| Glossary | `internal/handlers/glossary/`; `pages/glossary.templ` | Existing route; in-prose terms and reference presentation require expansion. |
| About | `internal/handlers/static/`; `pages/static.templ` | Existing static route; reference content and presentation required. |
| Statistics | `internal/handlers/statistics/`; `internal/repositories/statistics.go`; `pages/statistics.templ` | Existing route; accessible reference charts/tables and data coverage require expansion. |
| Visitor itineraries, builder, published view, slideshow | — | No production owner. The existing OpenSpec change supplies a candidate workflow only. |
| Timeline | — | No production handler, persistence model, route, or page. |
| Guided Tours and tour reading | — | No production handler, persistence model, route, or page. |

## Shared public capabilities

| Reference capability | Production owner | Status |
| --- | --- | --- |
| Layout, header, footer, feedback and keyboard mount | `internal/assets/templ/layouts/layout.templ`; `components/nav.templ`, `footer.templ`, `feedback.templ`, `keyboard.templ` | Existing shared mount; reference navigation grouping, footer preferences, and visual system require adoption. |
| Theme | `resources/css/style.pcss`; `resources/js/bootstrap.ts` | Existing interaction; reference semantic token and no-flash contract require alignment. |
| Bionic reading | `resources/js/bionic.ts`; page markup hooks | Existing helper; reference footer and cookie persistence contract require alignment. |
| Keyboard and palette | `resources/js/keyboard.ts`; `components/keyboard.templ`; `internal/handlers/keyboard/` | Existing foundation; registry and full screen/action coverage require expansion. |
| Feedback | Shared public layout | Floating control links directly to the public GitHub issue list; the in-application form is not the release entry point. |
| Cookie notice | `resources/js/cookieconsent.ts` | Existing library-driven behaviour; reference presentation requires alignment. |
| Image plate and viewer | `components/image.templ`; viewer setup in `resources/js/bootstrap.ts` | Existing viewer; reference rendition, keyboard, focus, and modal contract require replacement or adaptation. |
| Work cards, metadata, citations, fields, chips | Page-local markup plus `components/citation.templ` and `dialog.templ` | Partial/split ownership; reference requires reusable shared components. |
| Itinerary tray | — | No production owner. |
| Period music | Music collections exist; public handler registration is disabled in `internal/handlers/main.go` | No active public feature. |

## Production data and external hand-off

| Concern | Current owner | Required release action |
| --- | --- | --- |
| Synthetic seed | `internal/utils/seed/`; embedded synthetic assets | Retain only for development/test fixtures; remove it from release input selection. |
| Real source database | `wga-src/out/production/wga-src.sqlite` | Define release version and validation contract. |
| Real staged media | `wga-src/out/production/storage`; `cmd/wga-thumbnail-stager` | Copy the complete storage tree before preseeded records are saved. |
| Source-bundle manifest | `wga-src/cmd/wga-source-bundle-manifest` | Consume `out/production/source-bundle-manifest.json`, format version 1, before release seeding. |
| Thumbnail URL generation | `internal/utils/url/main.go` | Add the reference's missing named variants: 600 artwork fallback, 800, 1000, 1100, and 1600. |
| Artwork palette/signature | `wga-src/internal/importer/colour_profile.go` | Carry verified real-data profile fields into WGA for palette display and similarity queries. |

## Test ownership

Existing Playwright coverage exists for home, artists, artwork search and records,
global search, statistics, Dual Mode, glossary, guestbook, static pages, postcards,
coverage is required for selections, timeline, tours, itineraries, real-data release
seeding, accessible viewer behaviour, screenshot comparison, and assistive-technology
acceptance.

## Delivery implications

1. Existing feature owners are extension points, not evidence of reference parity.
2. Selections, timeline, tours, itineraries, tray, and active period music require new
   feature ownership before their pages can be implemented.
3. Shared visual work must precede page adoption to avoid independently recreating
   layout, keyboard, plate, dialog, card, and preference behaviour.
4. The real database and its staged storage are a release prerequisite for every image-
   and palette-dependent route.

## Existing OpenSpec reconciliation

| Existing change | Review result | Release disposition |
| --- | --- | --- |
| `add-visitor-itineraries` | Its anonymous draft, moderation, expiry, and purge model supports the release after adding all supported artwork entry points, the fifteen-stop limit, one-year expiry, and slideshow requirement. | Updated in place; its feature workflow remains an implementation input. |
| `add-artwork-relationship-paths` | Its canonical relations support curation, but its public generic-path presentation differs from the reference. | Updated in place: the first release exposes only artist, collection/current museum, palette, and forty-year period bases. |
| `generate-pocketbase-thumbnails` | The change has no proposal, design, specifications, or tasks. `wga-src` already defines and stages the exact thirteen rendition variants. | Superseded for release planning by `collection-data-release` and `visual-thumbnail-delivery`; do not independently implement a second variant generator. |

The producer-owned source-bundle manifest is generated as the final `mise run run`
step. It records SHA-256 values for the production SQLite database, staged storage
