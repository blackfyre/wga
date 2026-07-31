## ADDED Requirements

### Requirement: Rams-inspired public visual system
The system SHALL render every public page, dialog, cookie surface, and error page with the hand-off reference's bone ground, ink content, blue accent, square controls, hairline rules, system-content type, monospace metadata type, and restrained reduced-motion-aware transitions.

#### Scenario: Public page uses the shared shell
- **WHEN** a visitor opens any public route
- **THEN** the page renders the shared responsive shell and its content uses the common public visual system.

#### Scenario: Reduced motion is requested
- **WHEN** a visitor has enabled reduced motion
- **THEN** public transitions do not delay content, navigation, dialog use, or feedback.

### Requirement: Responsive public navigation
The system SHALL provide the reference's branded public header, search affordance, navigation destinations, and responsive mobile navigation while keeping navigation links usable without JavaScript.

#### Scenario: Desktop visitor navigates the catalogue
- **WHEN** a desktop visitor selects Artists or Artworks from the public navigation
- **THEN** the browser opens the corresponding public route.

#### Scenario: Mobile visitor opens navigation
- **WHEN** a mobile visitor activates the navigation control
- **THEN** the navigation destinations become visible and keyboard-accessible.

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
The system SHALL display Vanilla CookieConsent using the reference's notice treatment while retaining the existing client-side necessary-consent category, persistence, and preferences behaviour.

#### Scenario: Visitor accepts necessary cookies
- **WHEN** a visitor accepts the available necessary-consent action from the redesigned notice
- **THEN** Vanilla CookieConsent persists consent and does not show the initial notice again according to its existing lifecycle.

#### Scenario: Visitor opens cookie preferences
- **WHEN** a visitor selects the cookie-preferences action
- **THEN** Vanilla CookieConsent opens its preferences interface without a server-side consent request.
