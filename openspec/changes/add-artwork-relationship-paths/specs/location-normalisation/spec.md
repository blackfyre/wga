## ADDED Requirements

### Requirement: Raw location preservation

The system SHALL preserve every imported artwork location value as raw source text independently of any classification or canonical relation derived from it.

#### Scenario: Canonical museum match

- **WHEN** an imported location is matched to a canonical museum
- **THEN** the original location text SHALL remain available with the artwork.

### Requirement: Private collection classification

The system SHALL recognise configured variants of private-collection location text as an internal private-collection classification without assigning a public museum or location relation.

#### Scenario: Private collection variant

- **WHEN** an imported location matches a configured private-collection variant
- **THEN** the artwork SHALL be classified internally as private collection and SHALL have no public location-derived connection.

### Requirement: Curated museum aliases

The system SHALL resolve a normalised location value to a canonical museum only through an exact match to a curator-maintained canonical name or alias.

#### Scenario: Known museum variation

- **WHEN** an imported location normalises to a configured museum alias
- **THEN** the artwork SHALL be related to that canonical museum.

#### Scenario: Unknown location variation

- **WHEN** an imported location has no configured canonical museum name or alias
- **THEN** the system SHALL leave the museum relation unset.

### Requirement: Fuzzy-match review candidates

The system SHALL use edit-distance comparison only to identify review candidates for unresolved locations and SHALL NOT create a canonical museum relation from a fuzzy match alone.

#### Scenario: Probable museum variation

- **WHEN** an unresolved location is close to a canonical museum name or alias
- **THEN** the unresolved-location report SHALL include the candidate and its comparison score for curator review.

#### Scenario: Ambiguous fuzzy match

- **WHEN** an unresolved location has tied or near-tied museum candidates
- **THEN** the system SHALL leave the location unresolved until a curator adds or confirms an alias.

### Requirement: Unresolved-location reporting

The system SHALL generate a reviewable report of unresolved raw location values, grouped by normalised value and ordered to support iterative alias refinement.

#### Scenario: Repeated unresolved value

- **WHEN** multiple artworks share the same unresolved normalised location value
- **THEN** the report SHALL show one grouped entry with its occurrence count.
