## Context

The public WGA application already has PocketBase, Templ, HTMX, catalogue and comparison foundations. The `wga-visual-overhaul` hand-off defines the complete intended release: visual system, public pages, shared interactions, and several capability extensions. Its `WGA Prototype.dc.html` is the visual source of truth; the standalone and Templ artefacts are reference implementations, not production code.

Existing unimplemented changes for itineraries, artwork relationships, and thumbnails are useful inputs but predate this release definition. The real `wga-src` production SQLite dataset and its staged PocketBase storage replace the embedded synthetic database as the release deployment input without changing WGA's existing development/test seed fallback.

## Goals / Non-Goals

**Goals:**

- Deliver every non-development public feature described by the visual-overhaul reference, in phased implementation work but as one release scope.
- Preserve server-rendered, bookmarkable public routes and progressively enhance them with HTMX and focused browser helpers.
- Serve the real data set through thirteen approved image profiles, with source-eligible downscales pre-generated and originals used whenever a profile would upscale.
- Give regular visitors coherent discovery and participation paths, and give scholars inspectable, citable, shareable record and comparison paths.
- Make visual, functional, responsive, keyboard, no-JavaScript, and assistive-technology acceptance explicit.

**Non-Goals:**

- Build a custom public administration interface; PocketBase remains the operational interface for managed content and moderation.
- Ship the prototype's development-only viewport frame switch to public visitors.
- Add accounts, collaborative itinerary editing, or an unbounded generic relationship graph.
- Preserve the embedded synthetic data set as a production fallback.

## Decisions

### Treat the visual reference as the public release contract

The release implements the whole non-development reference rather than selecting a subset of screens. Delivery tracks sequence dependent work—data and shared shell first, then discovery, records and scholar tools, participation, and release hardening—but do not reduce the release scope. Chrome at 390px, 834px, and 1440px is the pixel-comparison reference; other supported browsers must retain the same hierarchy, usable controls, and responsive composition without requiring identical font rasterisation. Every reference audit and release screenshot run records the visual-overhaul repository HEAD and dirty/clean status. The current accepted reference baseline is the clean commit `629089b6268a94b62276ff8769d66e0d2a896022` (`refinement round`, 25 August 2026); earlier comparisons without a captured SHA remain explicitly inferred. The `wga-visual-overhaul` repository is a strictly read-only input: this change may inspect and compare it but must not modify, regenerate, commit, or otherwise treat it as an implementation worktree.

### Implement reference visual primitives and exact composition once

The production theme uses the reference system-font stack without loading a webfont, the complete `--t-*` relative type scale, and explicit `.text-muted`, `.text-faint`, `.text-faint-2`, and `.border-control` roles backed by the existing Rams colour properties in both themes. Shared and page templates consume those roles instead of bare-pixel type substitutions or unranked fallback colours. The shared `PageHead`, Plate, dialog, and layout contracts remain the single production implementations; Guestbook, Glossary, Licences, and reference pages do not reproduce their structure locally. Dialog dismissal remains in the reference header rule, and a word replaces an icon glyph wherever the reference supplies textual control copy.

Where an immutable reference literal conflicts with a documented measurable rule, implementation pauses until the user selects the controlling WGA acceptance criterion. On 26 August 2026, the user selected exact immutable literals for the identified palette conflict. WGA therefore matches the 52 affected reference token locations and records the resulting 53 measured token/ground contrast exceptions explicitly; it does not change the external reference, represent those exceptions as passing checks, or relax unaffected accessibility requirements.

The itinerary tray is a dark-inverted shared surface with visible clear and builder actions. Tray visibility reserves bottom space for both `#mc-area` and the fixed toast container so notifications cannot appear behind it. Statistics supplies server-rendered chart-equivalent visual summaries as well as data tables when JavaScript is unavailable. Exact capability composition is part of acceptance: selections use section `21` and the responsive two-column/four-column work-card grid; Guided Tours uses the four reference facts; and Timeline exposes artists, works, movements, buildings, events, and music lanes while leaving any lane without approved data honestly empty or unavailable rather than fabricating content.

### Separate palette identity from light/dark scheme

Appearance has two independent axes. The palette axis contains `bone`, `classic`, `verdigris`, `gothic`, `renaissance`, `baroque`, `rococo`, `classical`, `impressionist`, `catppuccin`, and `tokyo`; the scheme axis contains `light` and `dark`. Their product resolves to the corresponding daisyUI theme without altering markup or layout. Each palette supplies the immutable reference's complete semantic role set, chart-series ramp, and Timeline-lane colour/foreground pairs. Literal interface colours outside those role definitions are not accepted. Exact clean-reference literals control the 52 identified locations where the reference's values and prose contrast floors conflict; those 53 measured pair failures are a declared acceptance exception, while all unaffected contrast and role checks remain mandatory.

The browser stores palette and explicit scheme separately in local storage for pre-paint resolution and in cookies so server-rendered preference controls can mark an honest state. The inline head resolver applies both axes before the stylesheet; an explicit scheme outranks the operating system, while an unset scheme continues following operating-system changes. `baroque` and `tokyo` are dark-only: both scheme resolutions use their dark build, LIGHT is disabled with an explanation, and the underlying stored scheme is not overwritten.

Palette, scheme, and Bionic Reading live together in a footer-opened preferences panel rather than separate expanding footer controls or a settings route. Its trigger states the active combination, palette choices are grouped by provenance, and each palette uses text plus a split paper/ink swatch so colour is not its sole identifier. With JavaScript unavailable the page follows the operating-system scheme and hides or disables manual controls honestly.

### Reconcile existing OpenSpec work instead of preserving it unchanged

The new user-story and reference requirements define the intended public outcome. Before implementation starts, existing itinerary, relationship-path, and thumbnail artefacts must be reviewed and amended or superseded at their affected requirements and task boundaries. Their data/workflow choices remain only if they implement the release contract. This avoids duplicating database designs while preventing outdated implementation plans from setting public behaviour.

### Keep feature ownership behind handlers and workflows

Pages and handlers remain framework adapters. Each capability owns its workflow, persistence rules, and state transitions; shared components consume explicit DTOs rather than persistence records. This preserves the modular-monolith boundary while allowing the common layout, keyboard layer, image plate, citations, dialogs, preferences, and itinerary tray to be rendered once.

### Use real data and staged media as one release input

`wga-src` provides the production SQLite database, PocketBase-compatible storage tree, and producer manifest. Its `pb_schema.json` defines the approved artwork delivery profiles: 120, 200, 400, 500, 600, 700, 800, 900, 1000, 1100, 1400, 1600, and 2000 pixels wide. These are possible target widths, not mandatory derivative files for every source. `wga-src` records original image dimensions and stages only targets narrower than the source; it never upscales. The manifest is a contract between `wga-src` and the release operator, not a new WGA runtime input. The selected release path remains manual: a versioned provisioning runbook requires the operator to validate the paired output, original dimensions, and source-eligible downscales, pre-populate the complete PocketBase storage tree, install the approved database at the existing `WGA_SEED_SQLITE_PATH`, and record reviewable evidence before authorising startup. This change does not add a repository release CLI or replace the existing external deployment mechanism. WGA's importer continues recording filenames for preseeded assets and also carries source dimensions into its records; it does not validate the manifest or copy storage, and no `WGA_SEED_MANIFEST_PATH` is introduced. Handlers resolve named profiles through the URL utility and use the original URL without a `thumb` query whenever the target is not narrower than the source, preventing PocketBase from generating an upscale on first request.

### Model selections and related works as separate experiences

Artist selections are source-backed editorial readings of a bounded set of an artist's works. Each receives a preview and a dedicated artist-and-selection route derived from the producer's deterministic selection identity; WGA does not infer a form from titles, paths, or artwork metadata. Related works remain a record-level discovery row. It exposes the reference's four selected bases—artist, collection, palette, and period—through shareable query state, even where the underlying curator model contains richer canonical relationships. Sparse results are explained rather than padded or inferred.

Related-work cards are a sample, not the complete holding. Each basis resolves at most eight candidates and presents the four closest-date works. Artist, collection, and period can map to complete artwork-search filters, so their rows expose counted server-generated `FIND MORE` links; palette remains ranking-only because a palette-neighbour relation is not a stable catalogue facet.

### Treat catalogue facets as bounded server-owned views

Artwork facets are independently collapsible server-rendered groups with explicit summaries and active-count state. Collection selection uses `venue`; collection-name lookup uses `venue_q`. The handler owns collection counts, holding-size order, the forty-option cap, and the hidden-holdings note; templates do not recompute or imply a complete list. The year range uses the shared inline range presentation. Ordinary GET forms and URLs remain the source of truth underneath HTMX enhancement.

### Preserve supplied artist identity forms and evidence-backed reproduction metadata

Artist DTOs carry both the encyclopaedic filing form and the supplied short form. Indexes, results, citations, and artwork bylines use filing form; breadcrumbs and prose labels use short form. Public templates do not parse, reverse, or reconstruct names, and artist/date labels use a middot.

Artwork reproduction metadata follows the latest evidence boundary: display dimensions, format, weight, and the file link when present. The production page does not show a source or licence field merely to state an unsupported or absent claim. Persisted source provenance may remain available for internal integrity and citation workflows, but it is not promoted into the reference reproduction block without a future accepted contract.

### Render sampled palettes through one accessible bar

Artwork records and Dual Mode share one Templ palette-bar component and DTO. An artwork record weights its bands by each recorded surface share; Dual Mode uses equal-width bands because its comparison panes already identify the palette as sampled data. Every band is a native button with an accessible name containing its colour name, share, and hex value. Hover, focus, and tap reveal that same value in a tooltip; edge bands align their tooltip to the bar edge so it does not overflow its pane. The artwork record supplies the single caption that explains the affordance and that sampling is indicative rather than a pigment analysis; the Dual Mode help tip supplies that context for its panes. No repeated palette-value legend is rendered beside the shared bar.

### Preserve progressive enhancement and URL state

Every public destination and meaningful state has a server-rendered URL. HTMX may replace feature-owned blocks, and browser helpers may provide palette lookup, viewer control, playback coordination, preference transforms, prefetch, and responsive disclosures, but their absence cannot block core browsing, filtering, reading, citation, or sharing.

### Derive the release timeline from approved collection chronology

The release timeline uses approved art-period spans and the creation-date ranges of published artworks to produce its named spans, density context, lane marks, counts, and record links. WGA carries the required date-span fields through the real-data hand-off and derives timeline presentation deterministically from those fields.

Historical-event entries are deferred until a later change supplies an approved source-backed dataset. The current timeline does not infer events or event prose from artwork comments, filenames, prototype examples, or external general knowledge.

### Make accessibility a release boundary, not decorative polish

The shared keyboard registry owns shortcut labels and routes. Dialogs use the native modal contract unless an approved exception supplies equivalent focus movement, background inertness, Escape/visible dismissal, and focus restoration. Focus changes only at an open-state transition: preference selections and other unrelated client updates preserve the open panel's focus and scroll position. The deliberate artwork viewer must meet that same contract; retaining a pointer-only or non-modal viewer is not acceptable. Layouts use the reference's relative type scale and are checked at enlarged default text, reduced motion, narrow reflow, and with JavaScript disabled.

### Define public-data lifecycle before exposing participation

Anonymous itineraries, postcards, and guestbook entries are feature-owned records with explicit validation, honeypot/rate-limit controls, visibility, expiry or retention outcomes, and redacted operational logging. Itineraries publish immediately to immutable expiring public tokens and honour the visitor's link-only or listed discovery choice; guestbook entries remain private until moderated; postcard recipient pages remain token-restricted. Feedback links directly to the public GitHub issue list and creates no WGA-owned feedback record. Direct postcard-provider delivery is persisted before the external effect and has idempotent retry/recovery semantics.

Postcard recipient bearer tokens use random 256-bit values. Only their lookup hashes remain in ordinary records; the recoverable send value is stored as an AES-256-GCM envelope in the locked delivery record using a centrally validated, versioned keyring. Workers decrypt it only while processing, verify it against the stored hash, and purge the envelope after successful delivery or final resolution. Key rotation rewraps live envelopes without changing recipient URLs.

Approved guestbook entries are intentional public archival records for as long as approval remains in force. PocketBase superusers own moderation and retention decisions. The submission surface discloses that archival outcome; withdrawing approval immediately removes the entry from public queries and irreversibly redacts its name, location, and free text. Unreviewed and rejected private entries expire after ninety days through the owned lifecycle job.

Anonymous rate limits use a centrally validated client-identity policy selected by `WGA_CLIENT_IP_SOURCE`. `direct` parses the socket peer and ignores forwarding headers. `railway` is the production Railway-edge contract: it requires Railway's edge marker, accepts only Railway's single `X-Real-IP` client-address header, and ignores `X-Forwarded-For`; the service's application port remains unreachable except through Railway's public edge. Production startup requires an explicit source while local development defaults to `direct`. The resolver neither persists nor logs the raw client address, and a missing or malformed trusted identity fails closed for anonymous writes. Period-music cards and the direct player route expose only songs whose song and composer records are both explicitly published.

### Test by risk and public outcome

Go tests cover workflows, routes, queries, migrations, data-import contracts, and scheduled lifecycle jobs. Browser tests cover critical regular-visitor and scholar journeys, keyboard/no-JS paths, and visual reference states. The support baseline is current and previous stable Chrome, Edge, Firefox, macOS Safari, iOS Safari, and Android Chrome; accessibility checks include keyboard-only use, NVDA with Firefox, VoiceOver with Safari on macOS and iOS, and TalkBack with Chrome on Android.

## Risks / Trade-offs

- [The release combines visual work with data and workflow changes] → Deliver in dependency-aware tracks and retain capability-level tests and acceptance evidence.
- [Existing OpenSpec artefacts conflict with the reference] → Reconcile their requirements and tasks before applying either plan.
- [Real data, source dimensions, or staged storage is incomplete] → Make manifest, staging, external-seed configuration, and source-eligible rendition verification mandatory runbook checks before the operator authorises WGA startup.
- [Manual provisioning is performed incorrectly or without traceability] → Require a completed evidence record containing source and installed hashes, destinations, availability checks, operator identity, and release identifier.
- [A source is smaller than its surface profile] → Use the original URL rather than requesting an allowed missing thumbnail, because PocketBase may otherwise generate an upscale on first request.
- [Large artwork files cause slow or expensive delivery] → Use only the defined source-eligible rendition per surface and reserve the 2000px profile for deliberate viewer use.
- [Anonymous public workflows are abused or retain personal data too long] → Validate, rate-limit, moderate, redact, and run explicit lifecycle jobs with documented retention outcomes.
- [Pixel-perfect claims vary across engines] → Use Chrome screenshots as the reference and require functional, accessible layout equivalence elsewhere.
- [The viewer library cannot meet modal accessibility requirements] → Replace or adapt it behind the shared viewer contract before release.
- [The reference Templ theme helper contains stale comments and a duplicated `else` in its current working source] → Port the accepted palette/scheme behaviour and tested token values, not the reference implementation verbatim.

## Migration Plan

1. Audit current routes, DTOs, data fields, OpenSpec changes, and `wga-src` output against the release capability matrix.
2. Add original image dimensions to the source hand-off and WGA records, then use the manual provisioning runbook to validate the producer manifest and every source-eligible downscale, pre-populate PocketBase storage, install the database at the existing `WGA_SEED_SQLITE_PATH`, and record evidence before startup.
3. Add shared visual, navigation, preference, keyboard, dialog, and media foundations without removing server-rendered route behaviour.
4. Deliver capability tracks with their migrations, workflows, route contracts, fixtures, and browser acceptance evidence.
5. Require the external seed path in the production provisioning record while preserving WGA's existing empty-path synthetic fixtures for focused development and tests.
6. Run release visual, accessibility, real-data, privacy/lifecycle, and rollback checks before promoting the release.

Rollback disables new public routes or presentation paths while preserving forward-compatible data. Destructive schema removals, real-data replacement, and external messages require backups and their feature-specific recovery plans.
