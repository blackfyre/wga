## MODIFIED Requirements

### Requirement: Visitor can send a work as a postcard
The system SHALL let an unauthenticated visitor start from the public postcard entry point or a published artwork, choose a published work, and compose a postcard on a dedicated, bookmarkable page. The composer SHALL collect sender details, one to five validated recipient addresses, a message, and an optional period-music inclusion setting; it SHALL work with and without JavaScript.

#### Scenario: Visitor starts from the postcard entry point
- **WHEN** a visitor follows a public postcard call to action without a selected work
- **THEN** the system SHALL provide a navigable path to select a published artwork before composition.

#### Scenario: Visitor starts from a published artwork
- **WHEN** a visitor chooses to send a published artwork as a postcard
- **THEN** the system SHALL open the composer with that artwork's identity and a usable path to change the selection.

#### Scenario: Visitor submits multiple valid recipients
- **WHEN** a visitor submits between one and five distinct valid recipient addresses
- **THEN** the system SHALL persist delivery work for every submitted recipient and SHALL not silently discard an address.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor selects an artwork, corrects the form, or submits a valid postcard without JavaScript
- **THEN** the system SHALL complete the same journey through ordinary navigable pages and form responses.

### Requirement: Postcard delivery and recipient reading are recoverable
The system SHALL track postcard delivery state per recipient, make repeated external delivery attempts idempotent, and provide the recipient a real public postcard URL rather than an email-only rendering. The notification email SHALL show the selected artwork image when available, identify the selected work and sender, describe the postcard message accurately, and provide the private postcard link. It SHALL show period music only when the sender opted in and a matching published work is available.

#### Scenario: Recipient follows a delivered postcard link
- **WHEN** a recipient opens a valid postcard URL
- **THEN** the application SHALL render the selected work, message, sender context, and any opted-in available period-music card as a public page.

#### Scenario: A recipient delivery cannot be completed
- **WHEN** a postcard delivery reaches a terminal failure after its permitted attempts
- **THEN** the system SHALL record that recipient's failed state without representing the postcard as successfully sent.

### Requirement: Postcard submission is actionable and single-intent
The system SHALL retain valid entered values and present a specific corrective outcome for all expected submission rejections, including invalid CAPTCHA, throttling, unavailable artwork, invalid recipients, and provider unavailability. One visitor send intent SHALL create no more than one postcard and one delivery work item per selected recipient.

#### Scenario: Visitor corrects a rejected submission
- **WHEN** a postcard submission is rejected before work is queued
- **THEN** the composer SHALL show an actionable explanation, preserve the non-sensitive entered fields, and allow correction without reopening a blank form.

#### Scenario: Visitor repeats a send intent
- **WHEN** a visitor double-submits or retries the same pending send intent after an interrupted response
- **THEN** the system SHALL return the existing postcard outcome and SHALL NOT create duplicate delivery work.

#### Scenario: CAPTCHA cannot be accepted
- **WHEN** CAPTCHA verification is rejected or unavailable
- **THEN** the system SHALL explain that the postcard was not queued and SHALL NOT consume the visitor's successful-send allowance.

### Requirement: Confirmation communicates actual delivery state
The system SHALL distinguish queued work from recipient delivery, identify the selected work and chosen music availability, and give the sender a safe route to create another postcard.

#### Scenario: Postcard is accepted for delivery
- **WHEN** the system persists valid postcard work before an external delivery attempt
- **THEN** the confirmation SHALL state that the postcard is queued rather than claiming that it has been sent.

## ADDED Requirements

### Requirement: Postcard composition supports accessible form completion
The composer SHALL keep its primary fields and submit action visible and usable at supported responsive widths, initial keyboard focus SHALL reach the first required field, and validation feedback SHALL be associated with the corrected form state.

#### Scenario: Visitor composes on a narrow or wide viewport
- **WHEN** a visitor opens the composer at a supported responsive viewport
- **THEN** artwork context, all required fields, and the submit action SHALL remain usable without horizontal clipping or a trapped scroll position.
