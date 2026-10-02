# Public Page Experience

## Purpose

Define the shared public visual system, responsive navigation, static content, generated table of contents, and cookie-consent behaviour.

## Requirements

### Requirement: Rams-inspired public visual system

The system SHALL render every public page, dialog, cookie surface, and error page with the hand-off reference's bone ground, ink content, blue accent, square controls, hairline rules, system-content type, monospace metadata type, and restrained reduced-motion-aware transitions.

#### Scenario: Public page uses the shared shell

- **WHEN** a visitor opens any public route
- **THEN** the page renders the shared responsive shell and its content uses the common public visual system.

#### Scenario: Reduced motion is requested

- **WHEN** a visitor has enabled reduced motion
- **THEN** public transitions do not delay content, navigation, dialog use, or feedback.

### Requirement: Responsive public navigation

The system SHALL provide the reference's branded public header, search affordance, navigation destinations, and responsive mobile navigation while keeping navigation links usable without JavaScript. From 720px the header SHALL place the logo and the search group (search field, SEARCH and the Go to keyboard cue) on one row, with the navigation destinations and MORE on a second row. SEARCH and the keyboard cue SHALL keep their natural width on a single line, and the search field SHALL take the remaining space without falling below 140px. The navigation row SHALL be a single line from 834px; where it cannot fit, it SHALL wrap without hiding destinations. The header SHALL NOT cause horizontal page overflow. The desktop wordmark SHALL render `WEB GALLERY OF ART` in the monospace face at `--t-14`, weight 600, 3px letter-spacing and a 1.15 line height, and the strapline SHALL read `EUROPEAN ART, 3rd CENTURY – EARLY 20th` in the monospace face at `--t-10` with 1px letter-spacing, the faint colour role and a 4px top margin. MORE SHALL read `MORE ▾` without the browser's native disclosure marker.

#### Scenario: Desktop visitor navigates the catalogue

- **WHEN** a desktop visitor selects Artists or Artworks from the public navigation
- **THEN** the browser opens the corresponding public route.

#### Scenario: Mobile visitor opens navigation

- **WHEN** a mobile visitor activates the navigation control
- **THEN** the navigation destinations become visible and keyboard-accessible.

#### Scenario: Desktop header at intermediate and wide widths

- **WHEN** a visitor opens a public page at 720px, 834px, 1079px, 1080px or 1440px
- **THEN** the keyboard cue is one line no taller than 32px, bottom-aligned with SEARCH on the logo's row, the search field is at least 140px wide, and the page has no horizontal overflow.

#### Scenario: Navigation row fits on one line

- **WHEN** a visitor opens a public page at 834px or wider
- **THEN** every navigation destination and MORE sit on a single line.

#### Scenario: Visitor reads the brand and the MORE control

- **WHEN** a visitor opens a public page at 720px or wider
- **THEN** the wordmark and strapline use the design's type values and MORE reads `MORE ▾` with no native disclosure marker.

### Requirement: Public pages preserve functional routes and enhanced navigation

The system SHALL present redesigned versions of home, artists, artworks, artwork records, postcards, inspiration, statistics, static pages, guestbook, contributors, feedback, and error pages without removing their existing public route behaviour.

#### Scenario: Enhanced navigation is unavailable

- **WHEN** a visitor follows a public navigation link with JavaScript disabled
- **THEN** the destination page loads as a complete server-rendered document.

#### Scenario: An HTMX page update occurs

- **WHEN** a visitor uses an enhanced public interaction
- **THEN** the documented page block updates without duplicating the shared layout or losing the applicable browser URL.

### Requirement: Reference static content uses the public presentation system

The system SHALL seed and render About and every reference destination without an owning data-backed feature through the existing static-content mechanism, using the reference's information architecture and public visual system.

#### Scenario: Visitor opens a reference static destination

- **WHEN** a visitor follows About or another reference static-content navigation destination
- **THEN** the application renders the configured static content in the redesigned public page layout.

#### Scenario: Fresh data is initialised

- **WHEN** the application initialises a fresh data directory
- **THEN** the reference static-content records required by public navigation are available.

### Requirement: Static content provides a generated table of contents

The system SHALL derive a hierarchical table of contents from each static page's `h2` and `h3` headings, preserving existing heading IDs and assigning stable unique IDs to headings without one.

#### Scenario: Static content contains headings

- **WHEN** a static page contains `h2` or `h3` headings
- **THEN** the rendered page includes contents links to their stable fragment identifiers in heading order and hierarchy.

#### Scenario: Static content repeats a heading

- **WHEN** a static page contains repeated headings without explicit IDs
- **THEN** each heading receives a unique fragment identifier and every contents link targets its corresponding heading.

### Requirement: Cookie consent retains Vanilla CookieConsent semantics

The system SHALL display Vanilla CookieConsent using the reference's notice treatment while retaining the existing client-side necessary-consent category, persistence, and preferences behaviour. The system SHALL also offer an optional `preferences` category, off by default. That category SHALL gate every remembered preference: the artwork search toolbar choices, the colour scheme, the palette, the bionic-reading aid, and the Study Board. The notice and the preferences interface SHALL state that this optional storage exists and name what it remembers. They SHALL also state that `DENY`, or switching the category off, deletes what it stored.

The notice SHALL offer exactly three keyboard-operable actions, in this reading and focus order:

- `ACCEPT ALL` accepts every category, including `preferences`.
- `DENY` accepts the necessary category only, leaves optional storage off, and deletes any stored copy.
- `PREFERENCES` opens the preferences interface.

The preferences interface SHALL offer `ACCEPT ALL`, `DENY`, and `SAVE PREFERENCES` alongside a per-category toggle.

Without `preferences` consent, each remembered preference SHALL apply to the current page only. The system SHALL neither read nor write its browser storage, and SHALL delete any stored copy. When consent is first granted, the system SHALL store the state the page currently shows. When the consent library confirms consent that already existed, the system SHALL leave the remembered state unchanged. When consent is withdrawn, the system SHALL delete every stored copy while the open page keeps its current state. A grant or withdrawal made in another tab SHALL govern later reads and writes in an already-open tab.

#### Scenario: Visitor accepts necessary cookies

- **WHEN** a visitor activates `DENY` in the notice
- **THEN** Vanilla CookieConsent persists consent to the necessary category only, every stored preference copy is deleted, and the initial notice is not shown again according to its existing lifecycle.

#### Scenario: Visitor accepts all categories

- **WHEN** a visitor activates `ACCEPT ALL` in the notice
- **THEN** Vanilla CookieConsent persists consent to every category, including `preferences`, and the scheme, palette, bionic-reading and Study Board state the page shows is stored.

#### Scenario: Visitor opens cookie preferences

- **WHEN** a visitor selects `PREFERENCES`
- **THEN** Vanilla CookieConsent opens its preferences interface without a server-side consent request, listing the strictly necessary category and the optional preference-storage category with the preferences it covers.

#### Scenario: Visitor withdraws preference storage

- **WHEN** a visitor who accepted `preferences` consent switches it off and saves
- **THEN** the stored scheme, palette, bionic-reading, Study Board and search toolbar copies are deleted, and the open page keeps its current appearance and board.

#### Scenario: Consent changes in another tab

- **WHEN** a visitor grants or withdraws `preferences` consent in one tab while another tab is open
- **THEN** the other tab's next preference change is stored only if consent is now granted, and nothing that tab reads comes from storage that the withdrawal deleted.

### Requirement: Enhanced text search preserves in-progress input

The system SHALL issue an enhanced public text-search request only after the visitor has paused typing for 500 ms and the field's value has changed since the last request from that field. On the artists, artworks, Dual Mode artist index, guestbook, and glossary search forms, the system SHALL let a newer request from the same form supersede one still in flight. When a result produced by that same form replaces the block containing it, the system SHALL keep the visitor's current free-text field values, focus, and caret position rather than rendering the server's echoed values. Results produced by any other interaction on the same block, and full page loads, SHALL render the free-text field values that correspond to the resulting URL state.

#### Scenario: Visitor types a query at a normal pace

- **WHEN** a visitor types a multi-character word into a free-text field on one of the affected search forms without pausing for 500 ms between keystrokes
- **THEN** the system issues no search request until the visitor pauses, and then issues a single request for the complete word.

#### Scenario: Visitor keeps typing while a result is in flight

- **WHEN** a visitor types further characters into a free-text field after a search request from that form has been issued and before its result is swapped in
- **THEN** the field still contains every typed character, keeps focus and caret position after the swap, and a subsequent request reflects the complete value.

#### Scenario: A newer search supersedes an older one

- **WHEN** a search request from one of the affected forms is still in flight and the same form issues a newer request
- **THEN** the older request's result does not replace the block after the newer request has been issued.

#### Scenario: Non-value key events do not search

- **WHEN** a visitor presses keys in a free-text search field that do not change its value, such as arrow or modifier keys
- **THEN** the system issues no search request.

#### Scenario: Visitor resets or navigates the block

- **WHEN** a visitor has typed text into a free-text field and then activates a reset, sort, view, letter, or pane navigation control that replaces the same block
- **THEN** the free-text fields show the values of the resulting state, so a reset clears the typed text.

#### Scenario: Visitor returns through browser history

- **WHEN** a visitor navigates back or forward to an earlier state of an affected search page
- **THEN** the free-text fields show the values of that earlier state.

#### Scenario: Visitor has JavaScript disabled

- **WHEN** a visitor submits an affected search form without JavaScript
- **THEN** the page loads as a complete server-rendered document whose free-text fields show the submitted values.

### Requirement: Enhanced navigation recovers from edge challenges

When an enhanced `GET` request receives an edge challenge response, identified by the `cf-mitigated: challenge` response header, the system SHALL NOT swap the challenge response into the page and SHALL load the requested URL as a full document, so the visitor can complete the challenge and reach the destination. The system SHALL NOT convert challenged non-`GET` requests into navigations.

#### Scenario: Navigation link is challenged

- **WHEN** a visitor follows an enhanced navigation link and the edge answers the enhanced request with a challenge
- **THEN** the browser loads the link's URL as a full page instead of leaving the current page unchanged

#### Scenario: Enhanced search request is challenged

- **WHEN** an enhanced search or filter `GET` request is answered with a challenge
- **THEN** the browser loads that request's URL, including its query, as a full page

#### Scenario: Ordinary error response

- **WHEN** an enhanced request receives a `403` or other error response without the challenge header
- **THEN** the existing error handling applies and no navigation occurs

#### Scenario: Challenged submission

- **WHEN** an enhanced non-`GET` request is answered with a challenge
- **THEN** the system does not navigate or resubmit, and the challenge response is not swapped into the page

### Requirement: Public presentation matches the complete release reference

The system SHALL render all non-development public routes, shared surfaces, dialogs, error states, light/dark themes, typography, colour roles, square controls, and responsive tiers according to clean visual-overhaul reference commit `016cf6f0e93e88ce173bff3f9e8a09d854e52d35`. Typography SHALL use the reference system-font stack without a webfont, SHALL use the complete 33-rung relative type scale rather than bare-pixel substitutes, and SHALL preserve the muted, faint, secondary-faint, control-border, and independent action-link hierarchy in both themes. The exact rem values SHALL be `--t-9:.6875`, `--t-95:.71875`, `--t-10:.75`, `--t-105:.78125`, `--t-11:.8125`, `--t-115:.84375`, `--t-12:.875`, `--t-125:.90625`, `--t-13:.9375`, `--t-135:.953125`, `--t-14:.96875`, `--t-15:1`, `--t-16:1.0625`, `--t-17:1.09375`, `--t-18:1.15625`, `--t-19:1.21875`, `--t-20:1.28125`, `--t-21:1.34375`, `--t-22:1.40625`, `--t-26:1.625`, `--t-27:1.6875`, `--t-28:1.75`, `--t-30:1.875`, `--t-32:2`, `--t-34:2.125`, `--t-36:2.25`, `--t-38:2.375`, `--t-40:2.5`, `--t-44:2.75`, `--t-46:2.875`, `--t-48:3`, `--t-52:3.25`, and `--t-56:3.5`. Shared composition SHALL use the reference eight-pixel spacing rhythm, 48px primary controls, two-rank section headers, and right-aligned directional calls to action.

#### Scenario: Reference viewport is rendered in Chrome

- **WHEN** a supported Chrome release renders a public route at 390px, 834px, or 1440px
- **THEN** the layout, hierarchy, spacing, and surfaces match the accepted reference for that viewport.

#### Scenario: Visitor enlarges text or changes theme

- **WHEN** a visitor renders a public route with enlarged default text or the dark theme
- **THEN** the reference type hierarchy and muted, faint, secondary-faint, and control-border distinctions remain visible without lost responsive steps or bare-pixel overrides.

#### Scenario: Release verifies the raised type scale

- **WHEN** release verification inspects public CSS and rendered typography at a 16px root
- **THEN** all 33 token values match the accepted reference, `--t-9`, `--t-10`, `--t-11`, `--t-15`, and `--t-16` render at 11px, 12px, 13px, 16px, and 17px respectively, nothing renders below `--t-9`, only the frame switch and keyboard hint use that smallest rung, visitor-readable controls use at least `--t-10`, running prose uses at least `--t-15`, and `--t-9`/`--t-95` text uses no colour rank below muted.

### Requirement: Artist names follow the reference filing convention

The system SHALL render artist headings, indexes, search results, citations, and artwork bylines in encyclopaedic filing form such as `VERMEER, Johannes`, while breadcrumbs and sentence labels SHALL use the supplied short form. Artist/date combinations SHALL use a middot separator, and mononyms SHALL remain standalone.

#### Scenario: Visitor encounters an artist across public surfaces

- **WHEN** an artist appears in an index, result, citation, artwork label, breadcrumb, or sentence
- **THEN** the surface uses the appropriate filing or short form consistently without reconstructing a name from display text.

### Requirement: Artist indexes use scalable source-backed filters

The system SHALL render SCHOOL and PERIOD as native select controls on the artist index and in each Dual Mode artist index. Both option lists SHALL be alphabetical and derived from shared source-backed vocabularies rather than separate hardcoded page lists. The school list SHALL include the complete approved vocabulary, including schools with no current holding, and the home-page school count SHALL derive from that same roster. Period options SHALL remain limited to periods authoritatively associated with artists until the broader art-period vocabulary has been reconciled to artist records.

#### Scenario: Visitor opens artist filters

- **WHEN** a visitor opens the artist index or either Dual Mode artist index
- **THEN** SCHOOL and PERIOD each occupy one native select, support platform keyboard and type-ahead behaviour, remain legible in every supported palette, and preserve the selected values in the server-rendered URL state.

#### Scenario: Approved school has no current holding

- **WHEN** the shared school vocabulary contains a school with no matching published artist record
- **THEN** the school remains available in the alphabetical select and the home-page school total remains consistent with the complete shared vocabulary.

### Requirement: Public preferences are available in the footer

The system SHALL expose one shared-footer trigger whose label states the currently applied palette, light/dark scheme, and reading mode, and SHALL open an accessible preferences panel containing those site-wide choices. The panel SHALL apply initial focus and invoker restoration only when its open state changes, preserving the visitor's focus and scroll position through unrelated preference updates. The system SHALL not present one ever-widening inline footer control per preference or claim a client-only control works when its required script is unavailable.

#### Scenario: Visitor chooses dark appearance

- **WHEN** a visitor explicitly selects DARK
- **THEN** subsequent rendered public pages use the dark half of the selected palette without a light-theme or wrong-palette flash.

#### Scenario: Visitor changes a preference in an open panel

- **WHEN** a visitor changes a preference after scrolling the open preferences panel
- **THEN** the update does not move focus or reset the panel's scroll position.

#### Scenario: Open preferences panel receives an unrelated update

- **WHEN** a client or HTMX lifecycle update leaves the preferences panel open
- **THEN** the update does not move focus or reset the panel's scroll position.

### Requirement: Palette and light/dark scheme are independent remembered choices

The system SHALL provide the eleven reference palettes `bone`, `classic`, `verdigris`, `gothic`, `renaissance`, `baroque`, `rococo`, `classical`, `impressionist`, `catppuccin`, and `tokyo`. Each palette SHALL reproduce the complete immutable-reference interface-role, chart-series, and Timeline-lane token set without changing layout.

Palette and light/dark scheme SHALL be stored and resolved independently, in the `wga-palette` and `wga-theme` browser local storage entries only. Either entry SHALL be read or written only while the visitor has accepted the optional `preferences` cookie-consent category. An explicit choice SHALL take precedence over the operating-system scheme, and an unset scheme SHALL continue to follow operating-system changes.

Without that consent, a choice SHALL apply to the current page, including its HTMX updates. The next full page load SHALL use the default palette and the operating-system scheme. When consent is first granted, the system SHALL store the palette shown and any scheme explicitly chosen on the page. No palette or scheme cookie, request-context state, server preference DTO, or server-selected appearance control state SHALL exist. The legacy `wga_theme` and `wga_palette` cookies SHALL be expired.

The inline client head resolver SHALL apply valid local choices before stylesheet rendering to prevent a wrong-palette or wrong-scheme flash, and SHALL consult them only while the consent record accepts `preferences`.

For this change, exact clean-reference token literals control where the reference's prose contrast guidance contradicts those literals. The 53 measured token/ground exceptions SHALL remain explicitly documented rather than hidden as passing contrast checks.

#### Scenario: Visitor changes palette without changing scheme

- **WHEN** a visitor with `preferences` consent selects a different palette while using DARK
- **THEN** the selected palette's dark build is applied and remembered without changing the stored DARK choice.

#### Scenario: Visitor selects a dark-only palette

- **WHEN** a visitor selects `baroque` or `tokyo`
- **THEN** the dark-only build is applied, the LIGHT choice is visibly disabled with a reason, and the visitor's stored scheme remains unchanged for restoration after leaving that palette.

#### Scenario: Visitor has not accepted preference storage

- **WHEN** a visitor without `preferences` consent chooses DARK and a palette, then reloads the page
- **THEN** the choices applied before the reload, nothing was written to local storage, and the reloaded page uses the default palette and the operating-system scheme.

#### Scenario: A stale choice remains without consent

- **WHEN** a page loads without `preferences` consent while `wga-palette` or `wga-theme` still holds a value
- **THEN** the head resolver ignores the value, and the stored entries are deleted.

#### Scenario: Visitor identifies a palette choice

- **WHEN** the preferences panel lists palettes
- **THEN** choices are grouped by provenance and identified by label plus a paper/ink split swatch rather than colour alone.

#### Scenario: Release verifies immutable palette literals

- **WHEN** release verification compares WGA palette roles with clean reference commit `016cf6f0e93e88ce173bff3f9e8a09d854e52d35`
- **THEN** every token matches the immutable reference literal, and the known 53 contrast-floor exceptions are reported as explicit accepted exceptions rather than altering the external reference or substituting undeclared colours.

#### Scenario: JavaScript is unavailable

- **WHEN** a visitor opens a public route without JavaScript
- **THEN** the page follows the operating-system light/dark scheme, core content remains available, and unavailable manual palette or scheme controls are not presented as working.

#### Scenario: Browser storage is unavailable

- **WHEN** local storage cannot be read or written
- **THEN** the page uses the default palette and operating-system scheme without consulting or creating an appearance cookie, remains usable, and does not expose a server-derived preference state.

### Requirement: Public visual styling is application-owned

The system SHALL implement the accepted public visual and interaction contract through WGA-owned semantic roles and shared presentation primitives. The distributed application SHALL NOT depend on or ship daisyUI, its Tailwind plugin, its theme mechanism, its component classes, or a compatibility API that preserves that vocabulary. Retaining Tailwind as a build-time utility and responsive-layout layer SHALL NOT change the accepted reference rendering or progressive-enhancement behaviour.

#### Scenario: Release inspects the frontend dependency and styling surface

- **WHEN** release verification inspects frontend dependencies, CSS inputs, generated assets, Templ classes, and browser helpers
- **THEN** no daisyUI dependency, plugin, generated style, theme contract, component class, or compatibility alias remains.

#### Scenario: Visitor uses a WGA-owned visual primitive

- **WHEN** a public route renders a colour role, action, field, table, dialog, card, link, or loading placeholder
- **THEN** the surface retains the accepted reference appearance, semantics, keyboard behaviour, responsive composition, and palette/scheme response without daisyUI.

### Requirement: Public modal surfaces follow an accessible modal contract

The system SHALL use a labelled modal surface with deliberate initial focus, background inaccessibility, a visible dismissal control positioned in the reference header rule, Escape dismissal where safe, reduced-motion support, and focus restoration to its invoker.

#### Scenario: Visitor closes a public dialog

- **WHEN** a visitor dismisses a feedback, shortcut, or other public modal
- **THEN** focus returns to the invoker and background controls were not reachable while the modal was open.

### Requirement: Shared actions expose complete keyboard semantics

The system SHALL use native links and buttons for visually interactive rows, cards, table-of-contents items, carousel controls, palette swatches, and dismissals. Any approved non-native exception SHALL provide equivalent focusability, Enter and Space activation, visible focus, role, state, and an accessible name. Glyph-only controls SHALL have accessible names. Public text, note, and select fields SHALL expose the palette-aware shared `:focus-visible` treatment; they SHALL NOT suppress the browser outline unless an equally visible replacement focus style is active.

#### Scenario: Keyboard visitor traverses an interactive surface

- **WHEN** a visitor reaches an interactive row, carousel, table of contents, or glyph-only control by keyboard
- **THEN** the control is focusable in a logical order, identifies its purpose and state, shows visible focus, and performs the same action available to a pointer visitor.

### Requirement: Record cards and workspace chips use state-appropriate pointer feedback

The system SHALL highlight an available record-opening work, artist, tour, relationship, Dual Mode, selection, or Timeline card with a faint background tint rather than reducing the card or reproduction opacity. The tint SHALL extend beyond the plate without shifting the card's grid track or surrounding layout. Reduced opacity SHALL represent an unavailable record state only. Available itinerary and Study Board chips SHALL strengthen their outline and gain the matching faint ink or accent tint on hover; already-added or capacity-exhausted chips SHALL retain their resting appearance and SHALL NOT allow activation to reach a record action beneath or around them.

#### Scenario: Pointer traverses cards and workspace actions

- **WHEN** a visitor hovers an available record card, an available workspace chip, and an already-spent workspace chip
- **THEN** the card reproduction remains undimmed behind a stable tint, the available chip provides border-and-tint feedback, the spent chip provides no false hover offer, and activating either chip does not open the record.

### Requirement: Fixed bottom surfaces share one measured stack

The system SHALL coordinate the Study Board tray, itinerary tray, cookie notice, toast container, keyboard hint, and page-content reservation through one measured bottom-stack contract. When multiple fixed surfaces are visible, they SHALL not overlap one another or obscure page content or focused controls.

#### Scenario: Cookie notice and itinerary tray are visible together

- **WHEN** both fixed surfaces are rendered and a toast is raised
- **THEN** the Study Board shelf and itinerary tray remain adjacent in their reference order, the notice and toast clear the combined stack, and the page retains sufficient bottom space to reach its final content.

### Requirement: Shared footer exposes community destinations

The system SHALL include the refreshed reference's labelled Mastodon and GitHub community links in the shared public footer as ordinary destinations that remain usable without JavaScript.

#### Scenario: Visitor follows a community destination

- **WHEN** a visitor activates either footer community link
- **THEN** the browser follows the labelled external destination without requiring a scripted interaction.

### Requirement: Shared footer returns visitors to the top of the page

The system SHALL place a plain `↑ BACK TO TOP` text link in the shared footer's bottom row beside the preferences control. The link SHALL be an ordinary in-page link to the shared page header, SHALL work without JavaScript, and SHALL leave scrolling to the browser's CSS scroll behaviour, including its reduced-motion handling. Apart from the feedback control, the system SHALL not render a floating back-to-top control.

#### Scenario: Visitor returns to the top from the footer

- **WHEN** a visitor at the bottom of a public page activates `↑ BACK TO TOP`
- **THEN** the browser moves to the page header without a scripted interaction.

### Requirement: Rich-text prose and links follow the design type scale

The system SHALL render running prose in sanitised rich text (artist biographies, artwork and selection commentaries, Dual Mode panes, static pages, guided tours, and postcards) at `--t-16` with a 1.7 line-height in the text colour role, unless the surface explicitly selects another rung of the type scale, in which case that rung SHALL apply. Links inside that prose SHALL use the accent colour role with a 1px underline offset clear of the descenders, SHALL NOT change the surrounding weight, and SHALL take the secondary accent on hover. Headings, lists, code, quotations and strong text inside rich text SHALL use rungs of the type scale and the palette's colour roles. Public templates and styles SHALL NOT use a type size outside the 33-rung scale, including Tailwind's named `text-lg`, `text-xl` and `text-2xl` sizes. The keyboard focus ring SHALL use the active palette's accent colour role in every palette and scheme.

#### Scenario: Visitor reads an artist biography, artwork commentary or static page

- **WHEN** a visitor opens an artist biography, an artwork commentary, or a static page at 390px, 834px or 1440px in the light or dark scheme at a 16px root
- **THEN** each prose paragraph renders at 17px with a 28.9px line-height in the text colour role, and each prose link renders in the accent colour with a 1px underline at the paragraph's weight.

#### Scenario: Surface selects a different prose rung

- **WHEN** a prose surface such as a Dual Mode pane declares `--t-15`
- **THEN** its paragraphs render at that rung rather than the default prose rung.

#### Scenario: Visitor focuses a control in a dark palette

- **WHEN** a visitor moves keyboard focus in any palette and scheme
- **THEN** the focus ring uses that palette's accent colour role.

### Requirement: In-page navigation scrolls smoothly and respects reduced motion

The system SHALL scroll smoothly to the destination of an in-page fragment link in the public document and inside Dual Mode pane scroll regions. When a visitor has requested reduced motion, the system SHALL NOT animate any of these scrolls.

#### Scenario: Visitor follows an in-page anchor

- **WHEN** a visitor without a reduced-motion preference activates an "ON THIS PAGE", table-of-contents, back-to-top or other fragment link to an element on the current page
- **THEN** the viewport scrolls progressively to that element, and the URL fragment updates.

#### Scenario: Reduced motion is requested for an in-page anchor

- **WHEN** a visitor who has requested reduced motion activates the same fragment link
- **THEN** the viewport moves to the element immediately, without intermediate scroll positions.

### Requirement: Page navigations start at the top without disturbing in-place updates

The system SHALL return the viewport to the top, instantly, after an enhanced navigation swaps the main area and pushes or replaces the browser URL. The system SHALL leave the scroll position unchanged for fragment and in-place updates and SHALL retain HTMX history scroll restoration.

#### Scenario: Visitor navigates from a scrolled page

- **WHEN** a visitor scrolled down a page follows a public navigation or record link that replaces the main area
- **THEN** the new page is shown from its top, without a scroll animation.

#### Scenario: Fragment or in-place update occurs

- **WHEN** global search, artwork-search filtering, in-block pagination, a Dual Mode pane, the itinerary tray, the study board shelf, a toast or a dialog updates
- **THEN** the viewport scroll position is unchanged.

#### Scenario: Visitor goes back

- **WHEN** a visitor uses browser Back after an enhanced navigation
- **THEN** the previous page is restored at its previous scroll position.

#### Scenario: Visitor opens a page section link

- **WHEN** a visitor opens a public URL with a fragment that identifies a section
- **THEN** the page lands on that section rather than at the top.
