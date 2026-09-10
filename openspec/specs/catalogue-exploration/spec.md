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
