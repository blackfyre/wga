## ADDED Requirements

### Requirement: Timeline exploration exposes a URL-addressable collection window
The system SHALL render the collection timeline with a date-window range, density context, six reference lanes for artists, works, movements, buildings, events, and music, named art-period spans, an entry count, and a URL representing the selected window. Each lane SHALL use only approved source-backed data and SHALL expose an honest empty or unavailable state when that source has not been supplied. The timeline SHALL derive approved results from art-period data and published artwork creation-date ranges.

#### Scenario: Visitor changes the timeline window
- **WHEN** a visitor adjusts or submits the timeline range
- **THEN** the server renders the matching window, readout, panels, and marks at a shareable URL.

#### Scenario: Historical-event data is unavailable
- **WHEN** no approved source-backed historical-event dataset has been supplied
- **THEN** the timeline preserves the six-lane reference composition, marks the unavailable buildings/events or other unsupported lanes honestly, and renders its approved art-period and published-artwork chronology without inventing entries or prose.

### Requirement: Timeline remains usable without JavaScript
The system SHALL provide range controls and record links that work as ordinary server-rendered form and link interactions when JavaScript is unavailable.

#### Scenario: Visitor explores a timeline without JavaScript
- **WHEN** a visitor submits a valid timeline range without JavaScript
- **THEN** the response shows the selected window and its published entries without requiring browser scale calculations.
