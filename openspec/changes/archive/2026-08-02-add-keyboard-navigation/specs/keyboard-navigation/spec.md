## ADDED Requirements

### Requirement: Discoverable public keyboard navigation

The system SHALL provide a keyboard help surface and public navigation shortcuts across supported public screens. A single server-rendered screen registry SHALL define each supported section's shortcut letter, two-digit catalogue number, label, and URL, and SHALL generate the shortcut data, help content, and palette section rows.

#### Scenario: Visitor opens shortcuts help

- **WHEN** a visitor activates the help shortcut or control
- **THEN** the system displays the available section, finding, and browsing actions and an accessible dismissal path.

#### Scenario: Visitor jumps to a registered section

- **WHEN** a visitor presses a registered unmodified section letter or completes a registered two-digit catalogue number within one second
- **THEN** the system navigates to that registered section URL.

#### Scenario: Editable controls preserve typing

- **WHEN** focus is in an input, textarea, select, or contenteditable element
- **THEN** the system SHALL not process navigation or traversal shortcuts, except that Ctrl/Cmd+K opens the palette and Escape blurs the field.

### Requirement: Responsive search and dismissal

The system SHALL provide keyboard search and dismissal behaviour that reflects the current responsive navigation surface.

#### Scenario: Visitor focuses desktop search

- **WHEN** a visitor presses `/` outside an editable control at a desktop or tablet tier
- **THEN** the system focuses the marked global search field.

#### Scenario: Visitor opens mobile navigation for search

- **WHEN** a visitor presses `/` outside an editable control at the narrow responsive tier
- **THEN** the system opens the primary-navigation disclosure containing search and navigation controls.

#### Scenario: Visitor dismisses transient keyboard state

- **WHEN** a visitor presses Escape outside an editable control
- **THEN** the system closes open keyboard dialogs and mobile navigation, removes the current caret, and leaves no stale keyboard selection.

### Requirement: Keyboard list traversal

The system SHALL let visitors move one visible opted-in public result list with a non-focus-stealing caret and open marked record rows. Each list SHALL declare its visual column count so traversal follows the rendered list or grid.

#### Scenario: Visitor traverses artwork results

- **WHEN** a visitor uses the caret navigation shortcut on an artwork result list
- **THEN** the current item displays the caret marker and can be opened with Enter.

#### Scenario: Visitor traverses a result grid

- **WHEN** a visitor presses ArrowDown or ArrowUp, or J or K, in a marked grid
- **THEN** the caret moves by the declared number of columns and remains at the first or last available row when it cannot move farther.

#### Scenario: Visitor traverses adjacent records

- **WHEN** a visitor presses ArrowLeft or ArrowRight in a marked list or grid
- **THEN** the caret moves one available record without wrapping across either boundary.

### Requirement: Command palette lookup

The system SHALL provide a command palette backed by a capped, rate-limited server suggestion endpoint. The palette SHALL render and filter registered sections without a network request, then render matching public artists before public artworks once the visitor enters at least two non-space characters.

#### Scenario: Visitor searches the palette

- **WHEN** a visitor enters a valid palette query
- **THEN** the system debounces the request, renders matching public records within the available palette capacity, and lets the visitor open a section or record with Enter.

#### Scenario: Palette request is bounded

- **WHEN** the palette requests record suggestions
- **THEN** the request includes the remaining result capacity and the endpoint enforces the minimum query length, maximum result count, public-record filtering, and per-client rate limit.

#### Scenario: Limiter expires inactive clients

- **WHEN** a client rate-limit window has expired
- **THEN** the limiter removes its inactive state during subsequent request handling so distinct past clients do not grow process memory indefinitely.

### Requirement: HTMX lifecycle safety

The system SHALL reset keyboard traversal state after an enhanced page update.

#### Scenario: Results are replaced

- **WHEN** HTMX replaces an opted-in result block
- **THEN** stale current-item state is removed, the keyboard layer re-reads the replacement list and screen registry, and traversal uses only the replacement content.
