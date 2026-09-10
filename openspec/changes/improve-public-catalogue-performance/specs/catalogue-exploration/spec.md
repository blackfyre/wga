## ADDED Requirements

### Requirement: Public catalogue reads avoid unnecessary catalogue-wide work

The system SHALL limit each public catalogue response to the data work required by that response, SHALL reuse bounded catalogue-wide projections where repeated derivation would otherwise scan the catalogue for each request, and SHALL preserve the existing catalogue results and navigation contract.

#### Scenario: Results-only interaction

- **WHEN** an enhanced artwork-search request asks for only the results fragment
- **THEN** the system loads the matching result count and page without loading filter-option projections that are absent from the returned fragment

#### Scenario: Artist availability is projected

- **WHEN** an artist index or Dual Mode pane determines whether displayed artists have published works
- **THEN** the system resolves availability from a reusable bounded projection without expanding every published artwork relation for that individual request

#### Scenario: Co-authored work remains available

- **WHEN** a displayed artist appears as any author of a published co-authored work
- **THEN** the artist remains available under the optimised lookup

#### Scenario: Concurrent cold projection requests

- **WHEN** concurrent requests require the same application-scoped projection before it has been loaded
- **THEN** the system performs one shared load and gives each successful requester an equivalent result

#### Scenario: Catalogue projection becomes stale

- **WHEN** a relevant artwork, artist, or collection record changes
- **THEN** the next catalogue request derives affected reusable projections from current persisted data rather than serving the stale projection

#### Scenario: Existing catalogue interaction is preserved

- **WHEN** a visitor filters, sorts, pages, resets, changes result view, or hands a result to Dual Mode
- **THEN** the resulting records, canonical URL state, fragment target, and progressive-enhancement behaviour remain consistent with the existing catalogue requirements
