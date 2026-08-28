## ADDED Requirements

### Requirement: Visitors can maintain an anonymous bounded itinerary draft
The system SHALL let a visitor add published artworks to one session-owned draft of at most fifteen stops, show the fixed dark-inverted draft tray after the first addition with `CLEAR` and `ARRANGE & NARRATE →` actions, preserve the draft across navigation and reload, and reserve enough bottom space that neither page content nor toast notifications are obscured by the tray. Add controls SHALL expose the reference `compact`, `row`, and `block` presentations: compact uses short labels, row is a 46px inline full-label action, and block is a 50px full-width action.

#### Scenario: Visitor adds a work
- **WHEN** a visitor adds a published work from a supported card, record, or dual-mode pane
- **THEN** the itinerary tray shows the draft count, clear and builder actions, the work appears once in the session-owned draft, and any resulting toast is fully visible above the tray.

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
