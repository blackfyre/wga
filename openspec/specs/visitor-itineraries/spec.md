# visitor-itineraries Specification

## Purpose

TBD - created by archiving change adopt-visual-overhaul. Update Purpose after archive.

## Requirements

### Requirement: Visitors can maintain an anonymous bounded itinerary draft

The system SHALL let a visitor add published artworks to one session-owned draft of at most fifteen stops, show the fixed dark-inverted draft tray after the first addition with `CLEAR` and `ARRANGE & NARRATE →` actions, preserve the draft across navigation and reload, and reserve enough bottom space that neither page content nor toast notifications are obscured by the tray. Add controls SHALL expose the reference `compact`, `row`, and `block` presentations: compact uses short labels, row is a 46px inline full-label action, and block is a 50px full-width action.

#### Scenario: Visitor adds a work

- **WHEN** a visitor adds a published work from an artwork-search grid card or dense row, another supported card, a record, or a dual-mode pane
- **THEN** the itinerary tray shows the draft count, clear and builder actions, the work appears once in the session-owned draft, and any resulting toast is fully visible above the tray.

#### Scenario: Visitor adds a search result without opening its record

- **WHEN** a visitor activates `ADD TO ITINERARY +` on an artwork-search result
- **THEN** the draft mutation completes without following the surrounding artwork link, and the control reports `IN ITINERARY ✓` or `ITINERARY FULL` as applicable.

### Requirement: Visitors can arrange and narrate a draft

The system SHALL provide a server-persisted builder that permits stop ordering, bounded narration, removal, and clear actions through validated POST mutations.

#### Scenario: Visitor reloads a draft builder

- **WHEN** a visitor reloads after arranging or narrating a draft
- **THEN** the saved stop order and permitted narration are restored.

### Requirement: Published itineraries are expiring public slideshows

The system SHALL publish a validated draft to an immutable public token with a stated expiry, render it as a one-stop-at-a-time slideshow, and remove expired records through an owned lifecycle job.

#### Scenario: Recipient opens a published itinerary

- **WHEN** a recipient follows a valid public itinerary URL
- **THEN** they can read each stop with arrow-key, Escape, and ordinary link navigation and can inspect its deliberate artwork plate.

### Requirement: Itinerary admission budgets are configurable

The system SHALL bound new itinerary drafts and successful itinerary publications per trusted client identity within a rolling one-hour window, using budgets loaded from deployment configuration. When no budget is configured, the draft budget and the publication budget SHALL each default to 3. A configured budget SHALL be a positive integer; the application SHALL refuse to start with a non-positive or non-integer value and SHALL report which setting is invalid. Exhausting a budget SHALL keep the existing rate-limited response and toast.

#### Scenario: Production defaults apply

- **WHEN** no admission budget is configured and a client creates a fourth new draft within an hour
- **THEN** the request is refused with the "created too many itineraries" response

#### Scenario: Raised budget for verification

- **WHEN** the draft budget is configured to 1000 and one client creates 20 drafts within an hour
- **THEN** every draft creation is admitted

#### Scenario: Invalid budget is rejected

- **WHEN** a budget setting is configured as `0` or a non-integer
- **THEN** the application does not start and names the invalid setting
