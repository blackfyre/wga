## ADDED Requirements

### Requirement: Record pages offer optional session-wide period music
The system SHALL offer an applicable period-music card on supported collection records without autoplay and SHALL use one named player window for the browser session. A card or direct player route SHALL expose a stored song only when both the song and its composer are explicitly published.

#### Scenario: Visitor starts music from a record
- **WHEN** a visitor activates a period-music card
- **THEN** the application opens or reuses the named player window and hands it the selected piece without starting an additional player.

#### Scenario: Visitor requests an unpublished song
- **WHEN** a visitor requests a direct player route for a song or composer that is not published
- **THEN** the application returns the public not-found state without exposing its metadata or media URL.

### Requirement: Music controls retain honest fallback behaviour
The system SHALL provide a normal link when JavaScript is unavailable and SHALL visibly report blocked pop-ups rather than silently failing.

#### Scenario: Browser blocks the player window
- **WHEN** a visitor activates a music card and the browser blocks its window
- **THEN** the originating record displays a notice explaining that playback could not be opened.
