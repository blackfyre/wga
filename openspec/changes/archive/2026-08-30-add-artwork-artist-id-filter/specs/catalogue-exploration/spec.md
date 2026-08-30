## MODIFIED Requirements

### Requirement: Live catalogue filtering

The system SHALL let visitors filter artworks by text, school, form, type, reference-defined date range controls, and an exact artist-record identifier in URL state, and SHALL render a result summary, matching works, and an empty state. The exact artist-record filter SHALL select published artworks that include the identified artist, including co-authored works, without adding an artist identifier control to the visible catalogue form.

#### Scenario: Visitor applies a text filter

- **WHEN** a visitor enters a catalogue query and submits or waits for the enhanced search trigger
- **THEN** the result block shows only matching works and the browser URL represents the active filter state.

#### Scenario: Visitor opens an artist holding

- **WHEN** a visitor follows `FIND MORE BY … IN THE ARTWORK SEARCH` from a public artist record
- **THEN** the artwork-search URL contains that artist's exact public record identifier and results contain published works related to that record only.

#### Scenario: Visitor refines an artist holding

- **WHEN** a visitor with an exact artist-record filter changes another catalogue filter, result view, sort order, page, or Dual Mode hand-off
- **THEN** the exact artist-record filter remains in the resulting URL and continues to constrain the results.

#### Scenario: Visitor resets filters

- **WHEN** a visitor activates the reset control while an exact artist-record filter is active
- **THEN** the identifier and all other active filters clear and the unfiltered catalogue state is shown.

#### Scenario: Unknown artist identifier

- **WHEN** a visitor opens artwork search with an artist identifier that relates to no published artworks
- **THEN** the system renders the existing honest no-matching-works state without substituting a name-based search.

#### Scenario: Legacy artist URL remains usable

- **WHEN** a visitor opens an existing name-based artist search URL
- **THEN** the system preserves its existing name-based search behaviour.

#### Scenario: Exact artist identifier takes precedence

- **WHEN** a visitor opens artwork search with both a name-based artist filter and an exact artist-record identifier
- **THEN** the system uses the exact identifier, omits the name-based value from the canonical URL, and renders only the exact artist's published works.

#### Scenario: JavaScript is disabled

- **WHEN** a visitor submits active catalogue filters without JavaScript, including an exact artist-record filter opened from an artist record
- **THEN** the server renders the matching result page at a shareable URL with the identifier retained.
