## MODIFIED Requirements

### Requirement: Visitors can arrange and narrate a draft

The system SHALL provide a server-persisted builder that permits stop ordering, bounded and sanitised narration, removal, and clear actions through validated POST mutations.

#### Scenario: Visitor reloads a draft builder

- **WHEN** a visitor reloads after arranging or narrating a draft
- **THEN** the saved stop order and permitted narration are restored.

### Requirement: Published itineraries are expiring public slideshows

The system SHALL publish a validated draft containing at least one stop to an immutable, unguessable public URL with a stated one-year expiry. The system SHALL render only published, unexpired, moderated itineraries as a fixed slideshow that loads one stop at a time, presents the ordered stops and narration, and provides server-rendered fallback navigation without exposing draft-session data. An owned lifecycle job SHALL purge expired itineraries and abandoned drafts.

#### Scenario: Visitor publishes a draft

- **WHEN** a visitor publishes a valid draft with at least one stop
- **THEN** the system returns a share URL and expiry for the published itinerary.

#### Scenario: Recipient opens a published itinerary

- **WHEN** a recipient follows a valid public itinerary URL
- **THEN** they can read the ordered stops and narration with arrow-key, Escape, and ordinary link navigation and can inspect each deliberate artwork plate without exposing the draft session.

#### Scenario: Itinerary expires

- **WHEN** an itinerary reaches its expiry time
- **THEN** its public URL no longer renders the itinerary and the purge workflow can remove its persisted data.
