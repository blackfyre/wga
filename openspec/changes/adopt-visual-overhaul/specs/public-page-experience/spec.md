## ADDED Requirements

### Requirement: Public presentation matches the complete release reference

The system SHALL render all non-development public routes, shared surfaces, dialogs, error states, light/dark themes, typography, colour roles, square controls, and responsive tiers according to clean visual-overhaul reference commit `c09d13a5f80cdc8a980c07673bc26a4cf0e7b8cc`. Typography SHALL use the reference system-font stack without a webfont, SHALL use the complete 33-rung relative type scale rather than bare-pixel substitutes, and SHALL preserve the muted, faint, secondary-faint, control-border, and independent action-link hierarchy in both themes. The exact rem values SHALL be `--t-9:.6875`, `--t-95:.71875`, `--t-10:.75`, `--t-105:.78125`, `--t-11:.8125`, `--t-115:.84375`, `--t-12:.875`, `--t-125:.90625`, `--t-13:.9375`, `--t-135:.953125`, `--t-14:.96875`, `--t-15:1`, `--t-16:1.0625`, `--t-17:1.09375`, `--t-18:1.15625`, `--t-19:1.21875`, `--t-20:1.28125`, `--t-21:1.34375`, `--t-22:1.40625`, `--t-26:1.625`, `--t-27:1.6875`, `--t-28:1.75`, `--t-30:1.875`, `--t-32:2`, `--t-34:2.125`, `--t-36:2.25`, `--t-38:2.375`, `--t-40:2.5`, `--t-44:2.75`, `--t-46:2.875`, `--t-48:3`, `--t-52:3.25`, and `--t-56:3.5`. Shared composition SHALL use the reference eight-pixel spacing rhythm, 48px primary controls, two-rank section headers, and right-aligned directional calls to action.

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

The system SHALL provide the eleven reference palettes `bone`, `classic`, `verdigris`, `gothic`, `renaissance`, `baroque`, `rococo`, `classical`, `impressionist`, `catppuccin`, and `tokyo`. Each palette SHALL reproduce the complete immutable-reference interface-role, chart-series, and Timeline-lane token set without changing layout. Palette and light/dark scheme SHALL be stored and resolved independently in browser local storage only, with an explicit choice taking precedence over operating-system scheme and an unset scheme continuing to follow operating-system changes. No palette or scheme cookie, request-context state, server preference DTO, or server-selected appearance control state SHALL exist. The inline client head resolver SHALL apply valid local choices before stylesheet rendering to prevent a wrong-palette or wrong-scheme flash. For this change, exact clean-reference token literals control where the reference's prose contrast guidance contradicts those literals; the 53 measured token/ground exceptions SHALL remain explicitly documented rather than hidden as passing contrast checks.

#### Scenario: Visitor changes palette without changing scheme

- **WHEN** a visitor selects a different palette while using DARK
- **THEN** the selected palette's dark build is applied and remembered without changing the stored DARK choice.

#### Scenario: Visitor selects a dark-only palette

- **WHEN** a visitor selects `baroque` or `tokyo`
- **THEN** the dark-only build is applied, the LIGHT choice is visibly disabled with a reason, and the visitor's stored scheme remains unchanged for restoration after leaving that palette.

#### Scenario: Visitor identifies a palette choice

- **WHEN** the preferences panel lists palettes
- **THEN** choices are grouped by provenance and identified by label plus a paper/ink split swatch rather than colour alone.

#### Scenario: Release verifies immutable palette literals

- **WHEN** release verification compares WGA palette roles with clean reference commit `c09d13a5f80cdc8a980c07673bc26a4cf0e7b8cc`
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

The system SHALL use native links and buttons for visually interactive rows, cards, table-of-contents items, carousel controls, palette swatches, and dismissals. Any approved non-native exception SHALL provide equivalent focusability, Enter and Space activation, visible focus, role, state, and an accessible name. Glyph-only controls SHALL have accessible names, and suppressing a browser outline SHALL require a visible replacement focus style.

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
