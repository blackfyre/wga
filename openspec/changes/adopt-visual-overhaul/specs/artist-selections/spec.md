## ADDED Requirements

### Requirement: Artist records distinguish curated selections from catalogue holdings
The system SHALL render selection previews only for artists with more than one published source-backed curated selection, and SHALL otherwise retain the artist's ordinary works presentation.

#### Scenario: Artist has curated selections
- **WHEN** a visitor opens an artist record with curated selections
- **THEN** each preview states its supplied display title, selected and catalogued counts, editorial lede, representative works, and dedicated selection route.

### Requirement: A selection is a citable editorial record
The system SHALL render each selection at a stable artist-and-selection route derived from the producer's deterministic selection identity, with reference section number `21`, its lede, commentary, selected works in the reference responsive two-column/four-column work-card grid, links to the artist's other selections, a route to the wider holding, and a citation for the editorial text.

#### Scenario: Scholar opens a selection
- **WHEN** a scholar follows a selection route
- **THEN** the page distinguishes the selected argument from the full catalogue holding, presents section `21` and the responsive two-column/four-column work-card grid, and provides a copyable citation for that selection.

### Requirement: Missing editorial commentary is represented honestly
The system SHALL not generate or imply selection commentary where none exists.

#### Scenario: Selection has no commentary
- **WHEN** a visitor opens a curated selection without supplied commentary
- **THEN** the page states that the commentary is unavailable and does not substitute generated prose.

### Requirement: Curated artist records provide an optional stable in-page index
The system SHALL render `ON THIS PAGE` only when an artist record has one or more curated selections. Its ordinary fragment links SHALL target Biography, every selection form, and Cite This Record, and each selection entry SHALL report its shown-work count. The optional period-music player and TOC SHALL share one rail that becomes sticky from the reference medium breakpoint, subtracts the measured bottom stack from its available viewport height, and scrolls internally when necessary. Below that breakpoint the rail SHALL remain in normal document flow. An artist without curated selections SHALL omit the one-section TOC; when additional uncatalogued-on-page holdings exist, its artwork-search action SHALL instead appear after the citation as the final onward link. Scroll-position highlighting MAY enhance the links but SHALL NOT be required for navigation.

#### Scenario: Visitor follows the artist record index
- **WHEN** a visitor activates an `ON THIS PAGE` entry with a pointer or keyboard
- **THEN** browser-native fragment navigation reaches the labelled artist-record section and remains functional without JavaScript.

#### Scenario: Artist has no curated selection

- **WHEN** an artist record contains biography, holdings, and citation but no curated selection
- **THEN** no `ON THIS PAGE` navigation is rendered, the rail does not contain decorative navigation for a single argument, and any eligible `FIND MORE BY … IN THE ARTWORK SEARCH →` action follows the citation.

#### Scenario: Sticky artist rail shares space with fixed workspaces

- **WHEN** a curated artist record has period music or a TOC and a Study Board or itinerary tray is visible
- **THEN** the combined rail remains within the viewport above the measured bottom stack and its own content scrolls without obscuring its links or player.
