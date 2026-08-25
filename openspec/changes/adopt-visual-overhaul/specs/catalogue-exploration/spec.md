## ADDED Requirements

### Requirement: Artwork search provides the release filter and sort contract
The system SHALL support artwork filtering by school, period, form, technique, and location; grid and dense-list results; URL-addressable filter state; and sorting by catalogue, date, artist, or title with an explicit direction label. Tone-keyword filtering is deferred until an authoritative source dataset is available.

#### Scenario: Scholar changes sort criterion
- **WHEN** a scholar selects a new artwork sort criterion
- **THEN** the result URL records the criterion, applies that criterion's default direction, and labels the resulting order rather than using an ambiguous ascending label.

### Requirement: Artwork records support scholarly examination
The system SHALL render a deliberate full reproduction plate, reproduction source/file/licence details, metadata, an image-derived palette, commentary, related-work bases, citation, and a full-size source link when available.

#### Scenario: Scholar examines an artwork record
- **WHEN** a scholar opens a published artwork
- **THEN** they can inspect its stated reproduction and metadata, copy its citation, and open the deliberate image viewer without using a grid thumbnail as a viewer trigger.

### Requirement: Dual Mode compares complete independent records
The system SHALL render two independently addressable record windows with independent history, index/filter state, image-size state, and link-routing state, and SHALL provide an explicit wide override for a visitor whose browser zoom triggers the narrow layout gate.

#### Scenario: Visitor shares a comparison
- **WHEN** a visitor changes either pane or its target routing and shares the resulting URL
- **THEN** another visitor opens the same two records and pane-routing state.
