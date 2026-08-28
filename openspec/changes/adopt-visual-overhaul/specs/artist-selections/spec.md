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
