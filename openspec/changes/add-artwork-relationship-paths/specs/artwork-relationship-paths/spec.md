## ADDED Requirements

### Requirement: Artwork relationship fields
The system SHALL maintain explicit artwork relations for primary author, co-authors, schools or workshops, subjects, series or altarpieces, original locations, current museums, techniques, and art periods. The primary-author relation SHALL accept one artist; the co-author and shared-concept relations SHALL support multiple assignments where the collection needs them.

#### Scenario: Curator assigns shared concepts
- **WHEN** a curator assigns a subject, series, technique, or museum to an artwork
- **THEN** the artwork SHALL retain the selected canonical records as relations rather than relying on display-text equality.

#### Scenario: Curator distinguishes authorship
- **WHEN** a curator assigns a primary author and one or more co-authors to an artwork
- **THEN** the stored relations SHALL preserve the primary-author distinction.

### Requirement: Legacy author migration
The system SHALL migrate an existing single-value artwork `author` relation to `primary_author` without losing the artist assignment. The migration SHALL report artworks with multiple legacy authors for review rather than silently selecting one.

#### Scenario: Single legacy author
- **WHEN** an existing artwork has exactly one legacy author
- **THEN** that artist SHALL become the artwork's primary author.

#### Scenario: Multiple legacy authors
- **WHEN** an existing artwork has more than one legacy author
- **THEN** the migration SHALL identify the artwork in a review report and SHALL NOT silently discard an author assignment.

### Requirement: Related-artwork connection paths
The public artwork experience SHALL derive related artworks from shared canonical relationship values and SHALL label every displayed connection with its shared path.

#### Scenario: Shared series
- **WHEN** two published artworks share a series or altarpiece relation
- **THEN** each artwork's related-artwork presentation SHALL identify the other work as sharing that series or altarpiece.

#### Scenario: Shared authorship
- **WHEN** two published artworks share a primary author or co-author
- **THEN** the related-artwork presentation SHALL identify the shared artist connection.

#### Scenario: Multiple connection paths
- **WHEN** two artworks share more than one canonical relation
- **THEN** the related-artwork presentation SHALL retain each distinct shared connection path.

### Requirement: Private collection exclusion
The public artwork experience SHALL NOT create or display a related-artwork connection based only on an artwork being in a private collection.

#### Scenario: Private collection classification
- **WHEN** two artworks are internally classified as being in private collections
- **THEN** neither artwork SHALL appear as related to the other on that basis.
