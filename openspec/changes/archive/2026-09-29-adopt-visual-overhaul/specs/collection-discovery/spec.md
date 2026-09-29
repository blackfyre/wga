## ADDED Requirements

### Requirement: Collection discovery presents the release information architecture
The system SHALL expose Artists, Artworks, Timeline, Dual Mode, Guided Tours, Itineraries, and Inspiration as primary collection destinations, and SHALL expose Statistics, Glossary, Guestbook, Postcards, and About through MORE and the footer in the same order.

#### Scenario: Visitor opens the primary navigation
- **WHEN** a visitor uses the public navigation at a supported responsive tier
- **THEN** the collection destinations and their responsive disclosure are visible, labelled, keyboard-usable, and link to real server-rendered routes.

### Requirement: Home gives an unfamiliar visitor clear routes into the collection
The system SHALL render the home page with the collection argument, work of the day, artist and artwork counts, recent additions, and distinct discovery routes.

#### Scenario: Visitor opens the home page
- **WHEN** a regular visitor opens the home route
- **THEN** they can identify the collection and navigate to artist browsing, artwork browsing, or guided discovery without entering a search term.

### Requirement: Inspiration offers a non-prescriptive collection entry
The system SHALL render a shuffled, linkable slice of published collection works and direct visitors to Guided Tours and Itineraries when they want a curated route.

#### Scenario: Visitor seeks inspiration
- **WHEN** a visitor opens Inspiration
- **THEN** the page presents published works as record links and distinguishes its exploratory set from editorial tours and visitor itineraries.

### Requirement: Reference destinations retain a complete public route
The system SHALL render About, Contributors, privacy/reference content, and public error states through the shared public presentation while preserving their server-rendered URLs. Reference destinations, Guestbook, Glossary, and Licences SHALL use the shared page-head composition, and Contributors SHALL use textual control copy rather than an icon glyph where a word fits.

#### Scenario: Visitor opens a reference destination without JavaScript
- **WHEN** a visitor follows a reference-page route with JavaScript unavailable
- **THEN** the complete page content and applicable navigation render without a client-side dependency.

### Requirement: Statistics retain equivalent chart meaning without JavaScript
The system SHALL render a server-produced visual summary for each Statistics chart when JavaScript is unavailable, in addition to the accessible equivalent data tables.

#### Scenario: Visitor opens Statistics without JavaScript
- **WHEN** JavaScript is unavailable on the Statistics route
- **THEN** each chart's categories, relative values, caption, and corresponding table remain perceivable without a canvas script.
