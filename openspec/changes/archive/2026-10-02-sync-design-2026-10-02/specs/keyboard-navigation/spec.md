## ADDED Requirements

### Requirement: Keyboard hint bar is permanent on desktop pointers

The system SHALL render the keyboard hint bar along the bottom edge of every shared-layout public page from the first paint whenever the device reports a hovering, fine pointer, without waiting for a key press or any other keyboard use. The system SHALL hide the bar on devices without a hovering, fine pointer. The bar SHALL appear without a slide-in or other entrance animation, and its contents SHALL be unchanged. A full-viewport mode surface, such as Dual Mode or the itinerary viewer, covers the bar together with the rest of the shared page chrome (FEEDBACK and the trays) while it is open.

#### Scenario: Desktop visitor loads a page

- **WHEN** a visitor with a mouse or trackpad loads a public page and has not pressed any key
- **THEN** the keyboard hint bar is visible at the bottom edge, 30px tall, and the bottom-stack height already includes it.

#### Scenario: Touch visitor loads a page

- **WHEN** a visitor on a touch-only device loads a public page
- **THEN** the keyboard hint bar is not displayed and does not contribute to the bottom-stack height.
