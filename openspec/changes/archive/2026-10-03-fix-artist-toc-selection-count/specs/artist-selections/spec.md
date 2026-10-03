# Spec Delta

## MODIFIED Requirements

### Requirement: Curated artist records provide an optional stable in-page index

The system SHALL render `ON THIS PAGE` only when an artist record renders selection previews, meaning it has more than one published source-backed curated selection. Its ordinary fragment links SHALL target Biography, every rendered selection form, and Cite This Record, and each selection entry SHALL report the number of works in that selection, not the number its preview shows, as `1 work` or `N works`. The optional period-music player and TOC SHALL share one rail that becomes sticky from the reference medium breakpoint, subtracts the measured bottom stack from its available viewport height, and scrolls internally when necessary. Below that breakpoint the rail SHALL remain in normal document flow. An artist with fewer than two published source-backed curated selections SHALL retain the ordinary works presentation and omit selection previews, selection forms, and the one-section TOC; when additional uncatalogued-on-page holdings exist, its artwork-search action SHALL instead appear after the citation as the final onward link. Scroll-position highlighting MAY enhance the links but SHALL NOT be required for navigation.

#### Scenario: Visitor follows the artist record index

- **WHEN** a visitor activates an `ON THIS PAGE` entry with a pointer or keyboard
- **THEN** browser-native fragment navigation reaches the labelled artist-record section and remains functional without JavaScript.

#### Scenario: Artist has no curated selection

- **WHEN** an artist record contains biography, holdings, and citation but no curated selection
- **THEN** no `ON THIS PAGE` navigation is rendered, the rail does not contain decorative navigation for a single argument, and any eligible `FIND MORE BY … IN THE ARTWORK SEARCH →` action follows the citation.

#### Scenario: Artist has exactly one curated selection

- **WHEN** an artist record has exactly one published source-backed curated selection
- **THEN** the record retains its ordinary works presentation, renders no selection preview or selection form, omits `ON THIS PAGE`, and places any eligible `FIND MORE BY … IN THE ARTWORK SEARCH →` action after the citation.

#### Scenario: Sticky artist rail shares space with fixed workspaces

- **WHEN** a curated artist record has period music or a TOC and a Study Board or itinerary tray is visible
- **THEN** the combined rail remains within the viewport above the measured bottom stack and its own content scrolls without obscuring its links or player.

#### Scenario: A selection holds more works than its preview shows

- **WHEN** an artist record shows a selection of seven works whose preview shows four
- **THEN** that selection's `ON THIS PAGE` entry reads `· 7 works`, and a selection of one work reads `· 1 work`
