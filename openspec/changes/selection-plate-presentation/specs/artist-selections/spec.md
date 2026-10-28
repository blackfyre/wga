## MODIFIED Requirements

### Requirement: Artist records distinguish curated selections from catalogue holdings

The system SHALL render selection previews only for artists with more than one published source-backed curated selection, and SHALL otherwise retain the artist's ordinary works presentation. Each preview's representative works SHALL use the same selection plate grid and selection plates as the dedicated selection page.

#### Scenario: Artist has multiple curated selections

- **WHEN** a visitor opens an artist record with more than one published source-backed curated selection
- **THEN** each preview states its supplied display title, selected and catalogued counts, editorial lede, representative works, and dedicated selection route.

#### Scenario: Preview plates match selection page plates

- **WHEN** a visitor views a selection preview on the artist record and then the dedicated page for that selection at the same viewport width
- **THEN** the plate image boxes in both places have the same width and height.

### Requirement: A selection is a citable editorial record

The system SHALL render each selection at a stable artist-and-selection route derived from the producer's deterministic selection identity, with reference section number `21`, its lede, commentary, selected works in the selection plate grid, links to the artist's other selections, a route to the wider holding, and a citation for the editorial text. The selection plate grid SHALL show two equal columns below the reference medium breakpoint and, from that breakpoint, fixed-width plate columns that fill the available width, so plate size depends on the viewport width and not on the width of the surrounding page column.

#### Scenario: Scholar opens a selection

- **WHEN** a scholar follows a selection route
- **THEN** the page distinguishes the selected argument from the full catalogue holding, presents section `21` and the selection plate grid, and provides a copyable citation for that selection.

## ADDED Requirements

### Requirement: Selection plates show the stored holding location

A selection plate on the selection page or in an artist record's selection preview SHALL show the work's stored current holding name as a separate text entry after the title and artist; a long name MAY wrap, and it SHALL NOT be truncated. When the work has no stored current holding, or the stored holding cannot be resolved, the plate SHALL omit the entry and SHALL NOT infer a location from other text. Plates outside selections SHALL NOT show the entry.

#### Scenario: Work has a stored holding

- **WHEN** a selection includes a work whose current holding is stored as `Kunsthistorisches Museum, Vienna`
- **THEN** that work's plate shows `Kunsthistorisches Museum, Vienna` as a separate entry below the title and artist.

#### Scenario: Work has no stored holding

- **WHEN** a selection includes a work with no stored current holding
- **THEN** that work's plate shows no location entry and no placeholder.

#### Scenario: Plate outside a selection

- **WHEN** the same work appears on the Inspiration page
- **THEN** its plate shows no location entry.

### Requirement: Plates in one grid align

Public artwork plates in one grid SHALL have identical image boxes, and within one grid row the plates' trailing `VIEW WORK →` actions SHALL share one vertical position, as SHALL their itinerary actions, regardless of title, artist, or location length.

#### Scenario: Titles of different lengths share a row

- **WHEN** a row contains one plate with a one-line title and another with a three-line title
- **THEN** both plates' image boxes have the same size and their `VIEW WORK →` actions start at the same vertical position.
