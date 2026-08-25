## ADDED Requirements

### Requirement: Artwork records offer four explainable related-work bases
The system SHALL let visitors select BY ARTIST, SAME COLLECTION, SIMILAR PALETTE, or SAME PERIOD for related works, and SHALL represent the active basis in the artwork URL.

#### Scenario: Visitor changes the related-work basis
- **WHEN** a visitor selects a related-work basis on an artwork record
- **THEN** the record renders that basis's related works and a shared URL restores the same basis.

### Requirement: Related-work results state their connection and limits
The system SHALL render no more than four related works for the active basis, state the basis-specific connection, and explain sparse results with a link to the basis most likely to return records.

#### Scenario: Active basis has fewer than four records
- **WHEN** the active related-work query returns fewer than four published works
- **THEN** the remaining space explains the archive limit and does not fabricate or duplicate cards.

### Requirement: Palette similarity uses published image-derived data
The system SHALL calculate SIMILAR PALETTE from the real dataset's supported colour profile and SHALL exclude the current artwork and unpublished records.

#### Scenario: Artwork has no usable palette profile
- **WHEN** a visitor selects SIMILAR PALETTE for an artwork without a valid published colour profile
- **THEN** the page presents the documented sparse-result explanation rather than an unsupported similarity claim.
