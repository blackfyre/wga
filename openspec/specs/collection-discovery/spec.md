# collection-discovery Specification

## Purpose

TBD - created by archiving change adopt-visual-overhaul. Update Purpose after archive.

## Requirements

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

### Requirement: Statistics keys match their charts in every palette

The system SHALL render each Statistics chart key with exactly one swatch per series. Each swatch SHALL show the colour the chart uses for that series under the active palette and theme. A series that groups unattributed values ("Other") SHALL draw as a hatch in both the key and the chart. The chart SHALL draw no second legend of its own, and SHALL redraw with the new colours when the visitor changes the palette or the theme.

#### Scenario: Visitor reads the art-form key

- **WHEN** a visitor opens the Statistics route in any palette and theme
- **THEN** each art-form key row shows one swatch, and its colour equals the colour of the matching donut segment

#### Scenario: Visitor reads the school key

- **WHEN** a visitor opens the Statistics route with JavaScript available
- **THEN** each school key swatch matches the colour of that school's stacked segments, and "Other" is a hatch in both

#### Scenario: Visitor changes the palette or the theme

- **WHEN** a visitor changes the palette or the theme while the Statistics page is open
- **THEN** the charts redraw, and each key swatch still matches its chart series

### Requirement: Statistics copy states how the figures are produced

The system SHALL label the artworks chart as stacked by school and by the artist's birth period, and SHALL label each chart table as that chart's data. The footnote SHALL state how often the figures are recomputed in a way the implementation keeps true.

#### Scenario: Visitor reads the Statistics copy

- **WHEN** a visitor opens the Statistics route
- **THEN** the artworks chart reads "STACKED BY SCHOOL · BIRTH PERIOD OF ARTIST", each table caption ends "— DATA", and the footnote reads "RECOMPUTED HOURLY FROM PUBLISHED RECORDS."
