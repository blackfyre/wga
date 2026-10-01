## ADDED Requirements

### Requirement: Shared footer returns visitors to the top of the page

The system SHALL place a plain `↑ BACK TO TOP` text link in the shared footer's bottom row beside the preferences control. The link SHALL be an ordinary in-page link to the shared page header, SHALL work without JavaScript, and SHALL leave scrolling to the browser's CSS scroll behaviour, including its reduced-motion handling. Apart from the feedback control, the system SHALL not render a floating back-to-top control.

#### Scenario: Visitor returns to the top from the footer

- **WHEN** a visitor at the bottom of a public page activates `↑ BACK TO TOP`
- **THEN** the browser moves to the page header without a scripted interaction.
