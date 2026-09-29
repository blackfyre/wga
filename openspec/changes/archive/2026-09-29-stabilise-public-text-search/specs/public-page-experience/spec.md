# Spec Delta

## ADDED Requirements

### Requirement: Enhanced text search preserves in-progress input

The system SHALL issue an enhanced public text-search request only after the visitor has paused typing for 500 ms and the field's value has changed since the last request from that field. On the artists, artworks, Dual Mode artist index, guestbook, and glossary search forms, the system SHALL let a newer request from the same form supersede one still in flight. When a result produced by that same form replaces the block containing it, the system SHALL keep the visitor's current free-text field values, focus, and caret position rather than rendering the server's echoed values. Results produced by any other interaction on the same block, and full page loads, SHALL render the free-text field values that correspond to the resulting URL state.

#### Scenario: Visitor types a query at a normal pace

- **WHEN** a visitor types a multi-character word into a free-text field on one of the affected search forms without pausing for 500 ms between keystrokes
- **THEN** the system issues no search request until the visitor pauses, and then issues a single request for the complete word.

#### Scenario: Visitor keeps typing while a result is in flight

- **WHEN** a visitor types further characters into a free-text field after a search request from that form has been issued and before its result is swapped in
- **THEN** the field still contains every typed character, keeps focus and caret position after the swap, and a subsequent request reflects the complete value.

#### Scenario: A newer search supersedes an older one

- **WHEN** a search request from one of the affected forms is still in flight and the same form issues a newer request
- **THEN** the older request's result does not replace the block after the newer request has been issued.

#### Scenario: Non-value key events do not search

- **WHEN** a visitor presses keys in a free-text search field that do not change its value, such as arrow or modifier keys
- **THEN** the system issues no search request.

#### Scenario: Visitor resets or navigates the block

- **WHEN** a visitor has typed text into a free-text field and then activates a reset, sort, view, letter, or pane navigation control that replaces the same block
- **THEN** the free-text fields show the values of the resulting state, so a reset clears the typed text.

#### Scenario: Visitor returns through browser history

- **WHEN** a visitor navigates back or forward to an earlier state of an affected search page
- **THEN** the free-text fields show the values of that earlier state.

#### Scenario: Visitor has JavaScript disabled

- **WHEN** a visitor submits an affected search form without JavaScript
- **THEN** the page loads as a complete server-rendered document whose free-text fields show the submitted values.
