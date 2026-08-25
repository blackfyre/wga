## ADDED Requirements

### Requirement: Public presentation matches the complete release reference
The system SHALL render all non-development public routes, shared surfaces, dialogs, error states, light/dark themes, typography, colour roles, square controls, and responsive tiers according to the visual-overhaul reference.

#### Scenario: Reference viewport is rendered in Chrome
- **WHEN** a supported Chrome release renders a public route at 390px, 834px, or 1440px
- **THEN** the layout, hierarchy, spacing, and surfaces match the accepted reference for that viewport.

### Requirement: Public preferences are available in the footer
The system SHALL offer a remembered light/dark appearance choice and the release's reading preference in the shared footer without claiming a client-only control works when its required script is unavailable.

#### Scenario: Visitor chooses dark appearance
- **WHEN** a visitor explicitly selects DARK
- **THEN** subsequent rendered public pages use the dark presentation without a light-theme flash.

### Requirement: Public modal surfaces follow an accessible modal contract
The system SHALL use a labelled modal surface with deliberate initial focus, background inaccessibility, visible and Escape dismissal where safe, reduced-motion support, and focus restoration to its invoker.

#### Scenario: Visitor closes a public dialog
- **WHEN** a visitor dismisses a feedback, shortcut, or other public modal
- **THEN** focus returns to the invoker and background controls were not reachable while the modal was open.
