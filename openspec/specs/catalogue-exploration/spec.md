# Catalogue Exploration

## Purpose

Define the public artwork catalogue filtering, result presentation, paging, and Dual Mode hand-off behaviour.

## Requirements

### Requirement: Live catalogue filtering

The system SHALL let visitors filter artworks by text, school, form, type, reference-defined date range controls, and an exact artist-record identifier in URL state, and SHALL render a result summary, matching works, and an empty state. The exact artist-record filter SHALL select published artworks that include the identified artist, including co-authored works, without adding an artist identifier control to the visible catalogue form. While a valid exact artist-record filter is active, the system SHALL visibly identify the resolved public artist independently from the editable text query.

#### Scenario: Visitor applies a text filter

- **WHEN** a visitor enters a catalogue query and submits or waits for the enhanced search trigger
- **THEN** the result block shows only matching works and the browser URL represents the active filter state.

#### Scenario: Visitor opens an artist holding

- **WHEN** a visitor follows `FIND MORE BY … IN THE ARTWORK SEARCH` from a public artist record
- **THEN** the artwork-search URL contains that artist's exact public record identifier, the page visibly identifies that artist as the active scope, and results contain published works related to that record only.

#### Scenario: Visitor refines an artist holding

- **WHEN** a visitor with an exact artist-record filter changes the editable text query, another catalogue filter, result view, sort order, page, or Dual Mode hand-off
- **THEN** the exact artist-record filter remains in the resulting URL, continues to constrain the results, and remains visibly identified independently from the text query.

#### Scenario: Visitor resets filters

- **WHEN** a visitor activates the reset control while an exact artist-record filter is active
- **THEN** the identifier, visible artist scope, and all other active filters clear and the unfiltered catalogue state is shown.

#### Scenario: Unknown artist identifier

- **WHEN** a visitor opens artwork search with an artist identifier that relates to no published artworks
- **THEN** the system renders the existing honest no-matching-works state without substituting a name-based search.

#### Scenario: Artist identifier does not resolve publicly

- **WHEN** an exact artist-record identifier does not identify a published artist
- **THEN** the system does not display artist details for that identifier.

#### Scenario: Legacy artist URL remains usable

- **WHEN** a visitor opens an existing name-based artist search URL
- **THEN** the system preserves its existing name-based search behaviour without presenting it as an exact artist scope.

#### Scenario: Exact artist identifier takes precedence

- **WHEN** a visitor opens artwork search with both a name-based artist filter and an exact artist-record identifier
- **THEN** the system uses the exact identifier, omits the name-based value from the canonical URL, displays the resolved exact artist scope, and renders only the exact artist's published works.

#### Scenario: JavaScript is disabled

- **WHEN** a visitor submits active catalogue filters without JavaScript, including an exact artist-record filter opened from an artist record
- **THEN** the server renders the matching result page at a shareable URL with the identifier and visible artist scope retained.

### Requirement: Catalogue result views and paging

The system SHALL let visitors select the reference grid or list result presentation and navigate all result pages without losing active filters or the selected presentation.

#### Scenario: Visitor selects list view

- **WHEN** a visitor activates the list view control
- **THEN** matching works render in the list presentation and the selected state is exposed accessibly.

#### Scenario: Visitor visits another result page

- **WHEN** a visitor selects next or previous pagination
- **THEN** the requested page renders with the active filters and selected result view retained.

### Requirement: Catalogue search hands off to Dual Mode

The system SHALL preserve the active Dual Mode context when a visitor opens catalogue search from a selected Dual Mode pane and chooses an artwork result.

#### Scenario: Visitor replaces a selected Dual Mode pane

- **WHEN** a visitor opens artwork search for a Dual Mode target and chooses a work
- **THEN** the browser returns to `/dual-mode` with the chosen work in that target pane and the other pane and render-target state preserved.

### Requirement: Dual Mode retains full comparison controls

The system SHALL render Dual Mode in the reference visual system while retaining choice, lookup, pane-target selection, manual pane loading, copy, reverse, clear, and URL state behaviour.

#### Scenario: Visitor reverses panes

- **WHEN** a visitor activates the reverse comparison control
- **THEN** the left and right pane paths are exchanged and the resulting `/dual-mode` URL represents the new state.

#### Scenario: Visitor changes a pane target

- **WHEN** a visitor selects whether links from a pane open in the same or other pane
- **THEN** the selected target is exposed as current and survives subsequent comparison actions in the URL state.

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

### Requirement: Catalogue results highlight text search matches

In the grid and list result presentations, the system SHALL wrap the first case-insensitive occurrence of the active text search term in each artwork result's visible title and visible artist name in a `mark` element. The free-text query SHALL apply to both the title and the artist name. The title filter SHALL apply to the title, and when both are present it SHALL take precedence over the free-text query for the title. The artist text filter SHALL apply to the artist name, and when both are present it SHALL take precedence over the free-text query for the artist name, unless an exact artist-record filter is active. All highlighted and surrounding text SHALL be HTML-escaped. Attribute text (such as image alternative text, tooltips and accessible labels) SHALL remain unhighlighted plain text. The visible text of each result SHALL be the same as without highlighting.

#### Scenario: Free-text query matches the title

- **WHEN** a visitor requests `/artworks?q=madonna` and a result's title is "Madonna and Child"
- **THEN** the result's visible title renders "<mark>Madonna</mark> and Child"
- **AND** the image alternative text remains "Madonna and Child"

#### Scenario: Free-text query matches the artist name

- **WHEN** a visitor requests `/artworks?q=giotto` and a result's artist is "GIOTTO di Bondone" with a title that does not contain the term
- **THEN** the result's artist name renders "<mark>GIOTTO</mark> di Bondone" and its title has no `mark` element

#### Scenario: Highlight survives a results fragment swap

- **WHEN** a visitor types a text query and the debounced results fragment replaces the current results
- **THEN** the swapped-in results highlight the new query term

#### Scenario: Exact artist filter does not highlight the artist text filter

- **WHEN** a visitor requests `/artworks?artist_id=<id>&artist=rem`
- **THEN** no artist name is highlighted because of the artist text filter

#### Scenario: Search text is escaped

- **WHEN** a visitor requests `/artworks?q=<b>` and a result's title contains "<b>"
- **THEN** the response contains `<mark>&lt;b&gt;</mark>` and no unescaped markup from the query or title

#### Scenario: No text search

- **WHEN** a visitor filters only by school, form, type or date range
- **THEN** no result contains a `mark` element

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

The system SHALL render two independently addressable complete-record windows with independent history, index/filter state, and link-routing state, and SHALL provide an explicit wide override for a visitor whose browser zoom triggers the narrow layout gate. Each pane SHALL reuse the same accessible sampled-palette swatch control as an artwork record, with even-width bands and its existing palette help text instead of a duplicated value legend. Artwork panes SHALL include the record's location-note line, visible zoom affordance, citation access note, and one 1100px plate subject to the shared no-upscale rule. Dual Mode SHALL NOT expose, persist, or route an image-size choice.

An artist pane with source-backed curated selections SHALL render the same per-form selection headings, shown-versus-catalogued counts, ledes, representative works, and `OPEN SELECTION` actions as the artist record. Its pane-local selection view SHALL render the complete lede, commentary, selected works, holding note, sibling selections, and citation. Selection and sibling-selection actions SHALL update the pane already being read regardless of that pane's cross-window link-routing choice. A curated artist pane SHALL expose a contents list whose Biography, selection, and citation fragment identifiers are namespaced by pane; an ordinary flat-holding artist pane SHALL omit that decorative list.

#### Scenario: Visitor shares a comparison

- **WHEN** a visitor changes either pane or its target routing and shares the resulting URL
- **THEN** another visitor opens the same two records and pane-routing state.

#### Scenario: Scholar opens a curated selection inside one window

- **WHEN** a scholar activates `OPEN SELECTION` or a sibling selection from an artist pane
- **THEN** that same window renders the complete selection record, preserves the other window and both histories, and keeps every selection and citation destination unambiguous.

#### Scenario: Visitor opens a retired image-size URL

- **WHEN** a Dual Mode URL contains an obsolete pane image-size parameter
- **THEN** the canonical response URL or redirect removes that parameter while preserving both panes and every other pane-state value, no image-size control is shown, and the pane resolves the single 1100px plate or its original-file fallback.
