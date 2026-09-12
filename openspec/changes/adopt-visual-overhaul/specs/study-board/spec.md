## ADDED Requirements

### Requirement: The Study Board is an anonymous URL-shareable workspace

The system SHALL maintain an ordered, duplicate-free set of published artworks on a stable Study Board route. The canonical board URL SHALL contain the ordered artwork identifiers and SHALL be sufficient to share and restore the board without a server-persisted board record. A valid URL board SHALL outrank remembered browser-local state; otherwise the visitor's last browser-local board SHALL be restored. Invalid, missing, and unpublished identifiers SHALL be omitted without exposing their metadata.

#### Scenario: Recipient opens a shared board

- **WHEN** a recipient opens a board URL containing valid and invalid artwork identifiers
- **THEN** the published valid works appear once in URL order, invalid entries expose no metadata, and no account or stored board record is required.

#### Scenario: Visitor returns without a board URL

- **WHEN** a visitor previously maintained a board in the same browser and opens the Study Board without URL state
- **THEN** the remembered local board is restored and its canonical ordered URL is established.

### Requirement: Published works can be added from every reference surface

The system SHALL expose state-aware Study Board controls on artwork records, artwork-search grid cards and dense rows, both Dual Mode panes, eligible Timeline work selections, Inspiration cover works, and command-palette artwork results. A control SHALL report `ADD TO STUDY BOARD +` before addition and `ON STUDY BOARD ✓` after addition, SHALL not duplicate an existing work, and SHALL not trigger an enclosing record navigation action.

#### Scenario: Visitor adds a work from artwork search

- **WHEN** a visitor activates `ADD TO STUDY BOARD +` on an artwork-search result
- **THEN** the work is appended once, the canonical board URL and local continuation state are updated, the control reports `ON STUDY BOARD ✓`, and the artwork record is not opened.

### Requirement: The Study Board supports visual and metadata comparison

The system SHALL provide MATRIX and BOARD views over the same ordered works. MATRIX SHALL compare date, dimensions, medium, location, school, form, and type and SHALL mark `SAME` only when a value is known and equal for every work. BOARD SHALL show ordered artwork cards. Both views SHALL permit removal and reordering through native keyboard controls; drag-and-drop MAY provide an additional pointer interaction but SHALL NOT be the only reordering path.

#### Scenario: Visitor compares and reorders works

- **WHEN** a visitor changes view, moves a work earlier or later, or removes it
- **THEN** both views retain the resulting common order, the canonical URL is replaced without adding a navigation-history entry, and every action remains available by keyboard.

### Requirement: The Study Board has an honest transient lifecycle

The system SHALL describe the board as a scratch workspace with no title, narration, archive listing, publication state, server-persisted board record, or expiry schedule. It SHALL provide copy-link and clear actions. Outside the board route, a non-empty board SHALL render a fixed subordinate shelf with its count, work names, representative thumbnails, clear action, and `OPEN BOARD →` action.

#### Scenario: Visitor works with both fixed workspaces

- **WHEN** the Study Board and itinerary draft are both non-empty
- **THEN** the board shelf sits directly above the itinerary tray, both remain operable, and the shared bottom-stack reservation prevents either from obscuring content, notices, toasts, or focused controls.

### Requirement: A Study Board can safely become an itinerary draft

The system SHALL convert the first fifteen ordered board works into an itinerary draft while leaving the board unchanged. If an itinerary draft already contains works or narration, the system SHALL disclose the affected work and narration counts and require explicit destructive-replacement confirmation before invoking the itinerary-owned replacement workflow.

#### Scenario: Visitor converts over an existing narrated draft

- **WHEN** a visitor asks to turn a board into an itinerary while a narrated draft exists
- **THEN** the first activation warns that the draft and its narration will be replaced, conversion occurs only after explicit confirmation, no more than the first fifteen board works enter the new draft, and the Study Board remains unchanged.
