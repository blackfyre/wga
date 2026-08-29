## ADDED Requirements

### Requirement: Public presentation matches the complete release reference
The system SHALL render all non-development public routes, shared surfaces, dialogs, error states, light/dark themes, typography, colour roles, square controls, and responsive tiers according to the visual-overhaul reference. Typography SHALL use the reference system-font stack without a webfont, SHALL use the complete relative type scale rather than bare-pixel substitutes, and SHALL preserve the muted, faint, secondary-faint, and control-border hierarchy in both themes.

#### Scenario: Reference viewport is rendered in Chrome
- **WHEN** a supported Chrome release renders a public route at 390px, 834px, or 1440px
- **THEN** the layout, hierarchy, spacing, and surfaces match the accepted reference for that viewport.

#### Scenario: Visitor enlarges text or changes theme
- **WHEN** a visitor renders a public route with enlarged default text or the dark theme
- **THEN** the reference type hierarchy and muted, faint, secondary-faint, and control-border distinctions remain visible without lost responsive steps or bare-pixel overrides.

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
The system SHALL provide the eleven reference palettes `bone`, `classic`, `verdigris`, `gothic`, `renaissance`, `baroque`, `rococo`, `classical`, `impressionist`, `catppuccin`, and `tokyo`. Each palette SHALL reproduce the complete immutable-reference interface-role, chart-series, and Timeline-lane token set without changing layout. Palette and light/dark scheme SHALL be stored and resolved independently, with an explicit choice taking precedence over operating-system scheme and an unset scheme continuing to follow operating-system changes. For this change, exact clean-reference token literals control where the reference's prose contrast guidance contradicts those literals; the 53 measured token/ground exceptions SHALL remain explicitly documented rather than hidden as passing contrast checks.

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
- **WHEN** release verification compares WGA palette roles with clean reference commit `629089b6268a94b62276ff8769d66e0d2a896022`
- **THEN** every token matches the immutable reference literal, and the known 53 contrast-floor exceptions are reported as explicit accepted exceptions rather than altering the external reference or substituting undeclared colours.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor opens a public route without JavaScript
- **THEN** the page follows the operating-system light/dark scheme, core content remains available, and unavailable manual palette or scheme controls are not presented as working.

### Requirement: Public modal surfaces follow an accessible modal contract
The system SHALL use a labelled modal surface with deliberate initial focus, background inaccessibility, a visible dismissal control positioned in the reference header rule, Escape dismissal where safe, reduced-motion support, and focus restoration to its invoker.

#### Scenario: Visitor closes a public dialog
- **WHEN** a visitor dismisses a feedback, shortcut, or other public modal
- **THEN** focus returns to the invoker and background controls were not reachable while the modal was open.
