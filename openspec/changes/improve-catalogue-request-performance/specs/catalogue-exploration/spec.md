## MODIFIED Requirements

### Requirement: Public catalogue reads avoid unnecessary catalogue-wide work

The system SHALL limit each public catalogue response to the data work required by that response, SHALL reuse bounded catalogue-wide and reference-data projections where repeated derivation would otherwise scan stable data for each request, SHALL expose expensive artwork-facet stages through bounded privacy-safe tracing when configured telemetry is active, and SHALL preserve the existing catalogue results and navigation contract.

#### Scenario: Results-only interaction

- **WHEN** an enhanced artwork-search request asks for only the results fragment
- **THEN** the system loads the matching result count and page without loading filter-option projections that are absent from the returned fragment

#### Scenario: Artist availability is projected

- **WHEN** an artist index or Dual Mode pane determines whether displayed artists have published works
- **THEN** the system resolves availability from a reusable bounded projection without expanding every published artwork relation for that individual request

#### Scenario: Dual Mode reference data is projected

- **WHEN** Dual Mode needs the artist schools, art periods, and artist birth-year bounds used to build its panes
- **THEN** the system reuses one bounded application-scoped reference projection rather than reloading the same stable reference data for every request

#### Scenario: Co-authored work remains available

- **WHEN** a displayed artist appears as any author of a published co-authored work
- **THEN** the artist remains available under the optimised lookup

#### Scenario: Concurrent cold projection requests

- **WHEN** concurrent requests require the same application-scoped projection before it has been loaded
- **THEN** the system performs one shared load and gives each successful requester an equivalent result

#### Scenario: Catalogue projection becomes stale

- **WHEN** a relevant artwork, artist, collection, school, or period record changes
- **THEN** the next catalogue request derives affected reusable projections from current persisted data rather than serving the stale projection

#### Scenario: Artwork facet work is traced

- **WHEN** configured telemetry records a full artwork-search response
- **THEN** its trace distinguishes option loading, venue loading, school-count aggregation, and form-count aggregation with stable bounded stages

#### Scenario: Facet request contains visitor-controlled state

- **WHEN** an instrumented artwork-search request contains visitor-controlled filters or catalogue identifiers
- **THEN** facet-stage telemetry contains no filter values, query strings, URLs, record identifiers, result content, raw SQL, or arbitrary error text

#### Scenario: Existing catalogue interaction is preserved

- **WHEN** a visitor filters, sorts, pages, resets, changes result view, or hands a result to Dual Mode
- **THEN** the resulting records, canonical URL state, fragment target, and progressive-enhancement behaviour remain consistent with the existing catalogue requirements
