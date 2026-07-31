## ADDED Requirements

### Requirement: Live catalogue filtering
The system SHALL let visitors filter artworks by text, school, form, type, and reference-defined date range controls, and SHALL render a result summary, matching works, and an empty state.

#### Scenario: Visitor applies a text filter
- **WHEN** a visitor enters a catalogue query and submits or waits for the enhanced search trigger
- **THEN** the result block shows only matching works and the browser URL represents the active filter state.

#### Scenario: Visitor resets filters
- **WHEN** a visitor activates the reset control
- **THEN** all active filters clear and the unfiltered catalogue state is shown.

#### Scenario: JavaScript is disabled
- **WHEN** a visitor submits active catalogue filters without JavaScript
- **THEN** the server renders the matching result page at a shareable URL.

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
