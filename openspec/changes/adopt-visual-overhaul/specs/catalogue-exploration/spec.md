## ADDED Requirements

### Requirement: Artwork search provides the release filter and sort contract

The system SHALL support artwork filtering by school, period, form, technique, current collection, and an inline year range; grid and dense-list results; URL-addressable filter state; and sorting by title, artist, or date. School and form SHALL accept repeated query values with OR matching within the same facet and AND matching between different facets. The selected collection SHALL use `venue`, collection-name search SHALL use `venue_q`, and tone-keyword filtering remains deferred until an authoritative source dataset is available. Sorting SHALL use `sort=title|artist|date` and `dir=asc|desc`, default to title ascending, and treat obsolete catalogue-order or invalid sort input as title ascending.

#### Scenario: Visitor reviews or searches facets

- **WHEN** the artwork search renders filters
- **THEN** each facet is independently collapsible, title and school are initially open, valued facets reopen automatically, collapsed facets state their active summary, the heading reports the number of active filters, and the collection facet lists counted holdings sorted and capped at forty with an honest hidden-holdings note.

#### Scenario: Visitor combines school or form values

- **WHEN** a visitor selects multiple school or form values alongside other active filters
- **THEN** the result URL preserves every selected value as a repeated parameter, values in the same facet match with OR semantics, different facets match with AND semantics, each option count reflects the candidate's yield under all other active filters while ignoring its own facet's selected values, and the active-filter total counts each multi-select facet once regardless of its number of picks.

#### Scenario: Visitor reviews or expands a counted multi-select facet

- **WHEN** school or form has more values than its reference initial cap
- **THEN** the complete roster is ordered by count descending and label ascending for ties, the collapsed facet shows the first eight ranked options plus any selected option outside that cap, unavailable zero-result values remain as disabled stable rows, selected rows expose their selected mark and count without artificial rank promotion, and `SHOW ALL N`/`SHOW FEWER` expose or collapse the bounded full list without changing result filters.

#### Scenario: Visitor clears one multi-select facet

- **WHEN** a visitor activates the school or form facet's `CLEAR` action
- **THEN** only that facet's values are removed and every unrelated query, filter, view, and sort parameter is preserved.

#### Scenario: Scholar changes sort criterion

- **WHEN** a scholar selects a new artwork sort criterion
- **THEN** the result URL records that criterion with ascending direction, the selected control alone names `TITLE A–Z`, `ARTIST A–Z`, or `DATE EARLIEST`, the other two controls remain bare, and deterministic tie-breaking preserves stable pagination.

#### Scenario: Scholar reverses the active sort

- **WHEN** a scholar activates the currently selected TITLE, ARTIST, or DATE control
- **THEN** only its direction reverses, its label changes to `Z–A` or `LATEST` as applicable, and no separate reverse control or catalogue-order option is presented.

#### Scenario: Visitor follows an obsolete or invalid sort URL

- **WHEN** `sort=catalogue`, an unknown sort key, or an invalid direction reaches artwork search
- **THEN** the canonical result state and controls fall back to `sort=title&dir=asc` rather than exposing archive storage order.

### Requirement: Artwork records support scholarly examination

The system SHALL render a deliberate full reproduction plate, evidence-backed file type and pixel dimensions, metadata, an image-derived palette, commentary, related-work bases, citation, and a full-size file link when available. A conditional current-location note SHALL appear directly beneath the file link, and the citation SHALL follow the description and provenance content. The refreshed reproduction block SHALL not display file weight or unsupported reproduction-source or licence claims. An image-derived palette SHALL use a weighted sampled-colour bar rather than a repeated text legend: each swatch SHALL expose its available source-supplied name, share, and hex value on hover, keyboard focus, and tap. When no source-supplied name is recorded, the swatch SHALL omit the name while retaining its share and hex value. Tapping a swatch SHALL close any previously tapped palette tooltip, and tooltip width SHALL be independent of its associated band while remaining within the record surface. The record states that the sampling is indicative rather than a pigment analysis.

#### Scenario: Scholar examines an artwork record

- **WHEN** a scholar opens a published artwork
- **THEN** they can inspect its stated reproduction and metadata, copy its citation, and open the deliberate image viewer without using a grid thumbnail as a viewer trigger.

#### Scenario: Scholar examines an image-derived palette

- **WHEN** a published artwork has recorded palette data
- **THEN** its swatches are weighted by their recorded shares, provide their complete values without a competing text legend, and stay within the record surface at the first and last swatch.

### Requirement: Search updates expose reference loading states

The system SHALL render nine inert, `aria-hidden` skeleton rows or cards for artwork and artist result updates while an HTMX request is in flight. The visible result-count line SHALL remain the live status source, and repeated artwork metadata columns SHALL remain hidden in the dense list below the reference large breakpoint.

#### Scenario: Visitor changes an artwork or artist search

- **WHEN** an enhanced search request is in flight
- **THEN** the current result area presents the matching nine-item skeleton without announcing placeholder content, and the completed response replaces it while the count status communicates the result change.

### Requirement: Artwork search cards expose the reference metadata

The system SHALL project and render each artwork search result's filing-form artist name, date, school, form, and type without substituting technique for those fields. A grid card SHALL show the title, `ARTIST · DATE` on its first metadata line, and `SCHOOL · TYPE` on its second, followed by a full-width vertical itinerary/Study Board control block with consistent spacing. A dense-list row SHALL show `ARTIST · DATE` beneath its title, SHALL show separate School, Form, and Type columns at the reference large breakpoint, and SHALL place the same actions in a vertically stacked trailing block; the three repeated metadata columns SHALL be hidden below that breakpoint. Both presentations SHALL provide separate, state-aware `ADD TO ITINERARY +` and `ADD TO STUDY BOARD +` controls that do not activate the artwork-record link. Available controls SHALL strengthen their border and gain the appropriate faint tint on hover, while already-added or full controls SHALL remain visually inert without allowing activation to fall through to the record.

#### Scenario: Visitor compares grid and dense-list results

- **WHEN** the same artwork is rendered in grid and dense-list views
- **THEN** both views identify its title, filing-form artist, and date, the grid includes its school and type, and the large dense row includes its school, form, and type without displaying technique in place of a required value.

#### Scenario: Visitor adds a search result to a workspace

- **WHEN** a visitor activates the itinerary or Study Board control on a grid card or dense row
- **THEN** only the selected workspace changes, the control reports its resulting present or capacity state, the artwork-record link is not followed, and repeated activation does not add a duplicate.

### Requirement: Dual Mode compares complete independent records

The system SHALL render two independently addressable record windows with independent history, index/filter state, image-size state, and link-routing state, and SHALL provide an explicit wide override for a visitor whose browser zoom triggers the narrow layout gate. Each pane SHALL reuse the same accessible sampled-palette swatch control as an artwork record, with even-width bands and its existing palette help text instead of a duplicated value legend.

#### Scenario: Visitor shares a comparison

- **WHEN** a visitor changes either pane or its target routing and shares the resulting URL
- **THEN** another visitor opens the same two records and pane-routing state.
