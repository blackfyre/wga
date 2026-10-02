## MODIFIED Requirements

### Requirement: Responsive public navigation

The system SHALL provide the reference's branded public header, search affordance, navigation destinations, and responsive mobile navigation while keeping navigation links usable without JavaScript. From 720px the header SHALL place the logo and the search group (search field, SEARCH and the Go to keyboard cue) on one row, with the navigation destinations and MORE on a second row. SEARCH and the keyboard cue SHALL keep their natural width on a single line, and the search field SHALL take the remaining space: at least 140px wide from 720px when no classic scrollbar narrows the page, and shrinking rather than overflowing the row when one does. The navigation row SHALL be a single line from 834px, including when a classic scrollbar narrows the page; where it cannot fit, it SHALL wrap without hiding destinations. The header SHALL NOT cause horizontal page overflow. The desktop wordmark SHALL render `WEB GALLERY OF ART` in the monospace face at `--t-14`, weight 600, 3px letter-spacing and a 1.15 line height, and the strapline SHALL read `EUROPEAN ART, 3rd CENTURY – EARLY 20th` in the monospace face at `--t-10` with 1px letter-spacing, the faint colour role and a 4px top margin. MORE SHALL read `MORE ▾` without the browser's native disclosure marker.

#### Scenario: Desktop visitor navigates the catalogue

- **WHEN** a desktop visitor selects Artists or Artworks from the public navigation
- **THEN** the browser opens the corresponding public route.

#### Scenario: Mobile visitor opens navigation

- **WHEN** a mobile visitor activates the navigation control
- **THEN** the navigation destinations become visible and keyboard-accessible.

#### Scenario: Desktop header at intermediate and wide widths

- **WHEN** a visitor opens a public page at 720px, 834px, 1079px, 1080px or 1440px
- **THEN** the keyboard cue is one line no taller than 32px, bottom-aligned with SEARCH on the logo's row, the search field is at least 140px wide, and the page has no horizontal overflow.

#### Scenario: A classic scrollbar narrows the page

- **WHEN** a visitor whose platform draws a 17px non-overlay scrollbar opens a scrollable public page at 720px or 834px
- **THEN** the logo row does not overflow, the keyboard cue stays one line, and at 834px the navigation row is still a single line.

#### Scenario: Navigation row fits on one line

- **WHEN** a visitor opens a public page at 834px or wider
- **THEN** every navigation destination and MORE sit on a single line.

#### Scenario: Visitor reads the brand and the MORE control

- **WHEN** a visitor opens a public page at 720px or wider
- **THEN** the wordmark and strapline use the design's type values and MORE reads `MORE ▾` with no native disclosure marker.
