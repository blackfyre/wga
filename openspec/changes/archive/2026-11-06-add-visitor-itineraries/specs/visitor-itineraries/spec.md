## ADDED Requirements

### Requirement: Visitor itinerary drafts

The system SHALL maintain one session-owned itinerary draft of at most fifteen stops and SHALL let a visitor add, remove, order, and narrate published artwork stops from supported cards, records, and Dual Mode panes.

#### Scenario: Visitor adds an artwork

- **WHEN** a visitor adds a published artwork from a supported public surface
- **THEN** the system persists it in that visitor's draft and updates the itinerary tray.

### Requirement: Published itinerary sharing

The system SHALL publish a valid draft to an immutable, unguessable public URL with a stated one-year expiry.

#### Scenario: Visitor publishes a draft

- **WHEN** a visitor publishes a draft with at least one stop
- **THEN** the system returns a share URL and expiry for the published itinerary.

### Requirement: Safe public itinerary viewing

The system SHALL render published, unexpired, moderated itineraries without exposing draft-session data and SHALL provide a fixed slideshow that loads one stop at a time with server-rendered fallback navigation.

#### Scenario: Visitor opens a shared itinerary

- **WHEN** a visitor opens a valid published itinerary URL
- **THEN** the system renders its ordered stops and narration through the public slideshow without exposing the draft session.

### Requirement: Expiry and abuse controls

The system SHALL rate-limit publishing, sanitise narration, and purge expired itineraries and abandoned drafts.

#### Scenario: Itinerary expires

- **WHEN** an itinerary reaches its expiry time
- **THEN** its public URL no longer renders the itinerary and the purge workflow can remove its persisted data.
