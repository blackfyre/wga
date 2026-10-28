## MODIFIED Requirements

### Requirement: Dual Mode compares complete independent records

The system SHALL render two independently addressable complete-record windows with independent history, index/filter state, and link-routing state, and SHALL provide an explicit wide override for a visitor whose browser zoom triggers the narrow layout gate. Each pane SHALL reuse the same accessible sampled-palette swatch control as an artwork record, with even-width bands and its existing palette help text instead of a duplicated value legend. Artwork panes SHALL include the record's location-note line, visible zoom affordance, citation access note, and one 1100px plate subject to the shared no-upscale rule. Dual Mode SHALL NOT expose, persist, or route an image-size choice.

An artist pane with source-backed curated selections SHALL render one preview per selection with its heading, shown-versus-catalogued counts, lede, representative works, and `OPEN SELECTION` action; it SHALL NOT adopt the artist record's form-grouped selection index. Its pane-local selection view SHALL render the complete lede, commentary, selected works, holding note, sibling selections, and citation. Selection and sibling-selection actions SHALL update the pane already being read regardless of that pane's cross-window link-routing choice. A curated artist pane SHALL expose a contents list whose Biography, selection, and citation fragment identifiers are namespaced by pane; an ordinary flat-holding artist pane SHALL omit that decorative list.

#### Scenario: Visitor shares a comparison

- **WHEN** a visitor changes either pane or its target routing and shares the resulting URL
- **THEN** another visitor opens the same two records and pane-routing state.

#### Scenario: Scholar opens a curated selection inside one window

- **WHEN** a scholar activates `OPEN SELECTION` or a sibling selection from an artist pane
- **THEN** that same window renders the complete selection record, preserves the other window and both histories, and keeps every selection and citation destination unambiguous.

#### Scenario: Visitor opens a retired image-size URL

- **WHEN** a Dual Mode URL contains an obsolete pane image-size parameter
- **THEN** the canonical response URL or redirect removes that parameter while preserving both panes and every other pane-state value, no image-size control is shown, and the pane resolves the single 1100px plate or its original-file fallback.
