## Why

WGA's public application has working catalogue foundations, but its public journeys, visual language, image delivery, and real-data release path do not yet provide the complete archive experience defined in the accepted visual-overhaul reference. This release adopts that reference in full so regular visitors can discover and participate in the collection while scholars can inspect, compare, and cite it.

## What Changes

- Adopt the complete non-development public feature set and Rams-inspired visual system from clean `wga-visual-overhaul` commit `c09d13a5f80cdc8a980c07673bc26a4cf0e7b8cc`, including its system-font-only typography, readable relative type scale, named muted/faint/control-border and independent action-link roles, eleven independently remembered light/dark palette pairs, consistent 48px primary controls, eight-pixel spacing rhythm, tint-based record-card feedback, state-aware workspace-chip feedback, explicit keyboard/focus semantics, and reference-specific responsive composition, with pixel-accurate Chrome reference rendering at 390px, 834px, and 1440px viewports.
- Remove daisyUI completely and make the public visual system application-owned. Retain Tailwind as the build-time utility and responsive-layout layer, with WGA-owned semantic tokens and shared component styling derived from the authoritative prototype rather than a third-party theme or component vocabulary.
- Deliver the full public collection experience: home and navigation; artist, artwork, and selection records; counted multi-select artwork search with itinerary and Study Board actions; artwork and artist search loading states; inspiration; timeline; dual mode; the URL-shareable Study Board; glossary; statistics; guided tours; and reference pages.
- Build the release timeline from approved art-period spans and published artwork creation dates; defer external historical-event entries until a later source-backed change.
- Deliver visitor participation: postcards, guestbook signing and browsing, direct project issue-tracker feedback, anonymous itineraries, published itinerary slideshows, and the persistent itinerary tray.
- Deliver shared public interactions: keyboard navigation and command palette, accessible reference-positioned dialog controls and image viewer, native or keyboard-equivalent activation for every interactive row and carousel, glossary help, citation copying, sampled-palette swatches that disclose their values on hover, focus, and tap, a footer-opened preferences panel for independent palette, light/dark, and bionic-reading choices, constrained native period-music controls, cookie notice, responsive behaviour, coordinated fixed-bottom surfaces, tray-aware toast placement, equivalent no-JavaScript chart summaries, and the exact accepted artist index, selection, Timeline, Guided Tours, and itinerary-tray composition.
- Use the real `wga-src` production database and staged PocketBase storage as the release seed contract. Treat the thirteen authoritative widths as approved delivery profiles, pre-generate only source-eligible downscales, and serve the original whenever an assigned profile would upscale. The release operator follows a documented manual provisioning runbook to validate the producer manifest, pre-populate PocketBase storage, install the approved database at the existing `WGA_SEED_SQLITE_PATH`, and record evidence before startup; WGA does not gain a runtime manifest setting or storage-copy responsibility. Retain the embedded synthetic database only as a development/test fixture in the release workflow until separately removed.
- Reconcile existing but unimplemented OpenSpec changes with these user stories and reference requirements before implementation. In particular, retain their useful data/workflow designs only where they support the release behaviour.

## Capabilities

### New Capabilities

- `collection-discovery`: Home, inspiration, shared public navigation, and reference-page discovery routes.
- `artist-selections`: Curated, citable artist selections with preview and dedicated reading routes.
- `artwork-relationship-exploration`: Three-basis related-work discovery with explainable results and honest sparse-result states.
- `study-board`: A browser-local, URL-shareable workspace for comparing, ordering, and transferring published artworks into an itinerary draft.
- `timeline-exploration`: Server-rendered, URL-addressable collection timeline with progressive enhancement.
- `guided-tours`: Editorial, paged tours with contents, sources, and distinct tour-reading behaviour.
- `visitor-itineraries`: Anonymous drafts, narration, publishing, expiring sharing, and slideshow reading.
- `postcard-sharing`: Public postcard composition, delivery, confirmation, and recipient reading experience.
- `guestbook-participation`: Moderated guestbook signing and searchable public archive.
- `period-music`: Session-wide optional period-music playback associated with collection records.
- `collection-data-release`: Real-dataset seeding, staged media delivery, and release completeness verification.

### Modified Capabilities

- `public-page-experience`: Adopt the complete responsive public shell, application-owned visual system, navigation information architecture, preferences, and public-route presentation without daisyUI.
- `catalogue-exploration`: Expand catalogue filtering, sorting, record presentation, and complete two-pane comparison behaviour. Tone-keyword exploration is deferred to a later source-backed change.
- `visual-thumbnail-delivery`: Use every reference-defined delivery profile without upscaling and require source-eligible staged variants from the real data release.
- `keyboard-navigation`: Cover every release screen and on-page action with the reference keyboard contract.
- `public-feedback`: Route the floating feedback control directly to the public project issue tracker.
- `glossary-browsing`: Add in-prose glossary definitions and shared help-tip behaviour.
- `artist-portraits`: Align portrait surfaces and rendition use with artist records and selections.

## Impact

- Public Templ pages, shared layouts/components, CSS theme, browser helpers, handlers, feature workflows, PocketBase migrations, collections, routes, and release provisioning documentation.
- Frontend dependencies, Tailwind theme exposure, appearance resolvers, styling-contract tests, generated notices, and release assets change to remove daisyUI and its component/theme API while preserving the accepted visual and interaction contract.
- The existing itinerary, relationship-path, and thumbnail OpenSpec changes require review and amendment where their scope or assumptions differ from this release contract.
- `wga-src` is the real-data source and must deliver its production SQLite database, staged PocketBase storage tree, and producer manifest together; the release operator manually provisions that hand-off through WGA's existing external-seed contract and records the verification evidence, while handlers use complete thumbnail URLs rather than generate image variants on demand.
- Release acceptance covers Chrome visual comparison and functional/accessibility checks in current and previous Chrome, Edge, Firefox, Safari macOS/iOS, and Chrome Android, with defined keyboard, NVDA, VoiceOver, and TalkBack checks.
