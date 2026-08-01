## ADDED Requirements

### Requirement: Discoverable public keyboard navigation
The system SHALL provide a keyboard help surface and public navigation shortcuts that do not interfere with editable controls.

#### Scenario: Visitor opens shortcuts help
- **WHEN** a visitor activates the help shortcut or control
- **THEN** the system displays the available keyboard actions and an accessible dismissal path.

### Requirement: Keyboard list traversal
The system SHALL let visitors move a visible opted-in public result list with keyboard controls and open the current result.

#### Scenario: Visitor traverses artwork results
- **WHEN** a visitor uses the caret navigation shortcut on an artwork result list
- **THEN** the current item is visibly identified and can be opened with the keyboard.

### Requirement: Command palette lookup
The system SHALL provide a command palette backed by a capped, rate-limited server suggestion endpoint.

#### Scenario: Visitor searches the palette
- **WHEN** a visitor enters a valid palette query
- **THEN** the system renders matching public records and lets the visitor open one.

### Requirement: HTMX lifecycle safety
The system SHALL reset keyboard traversal state after an enhanced page update.

#### Scenario: Results are replaced
- **WHEN** HTMX replaces an opted-in result block
- **THEN** stale current-item state is removed and traversal uses the replacement list.
