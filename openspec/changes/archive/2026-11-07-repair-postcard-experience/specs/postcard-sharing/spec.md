## MODIFIED Requirements

### Requirement: Visitor can send a work as a postcard

The system SHALL let an unauthenticated visitor start from the public postcard entry point or a published artwork, choose a published work, and compose one postcard for it on a dedicated, bookmarkable page that works with and without JavaScript. The composer SHALL collect between one and five recipient addresses, a message, and sender details, with validation and abuse protection. The message field SHALL progressively enhance from an ordinary textarea to a Squire editor offering only bold, italic, bulleted-list, and numbered-list formatting, and Squire SHALL replace the existing Trix editor and dependency. The system SHALL normalise and deduplicate recipient addresses and SHALL enforce the five-address limit server-side. The compose form SHALL support adding and removing recipient rows through the existing submission endpoint with or without JavaScript, SHALL retain at least one row, and SHALL NOT offer or accept a sender-controlled period-music inclusion setting.

#### Scenario: Visitor submits a valid postcard

- **WHEN** a visitor submits a valid postcard form without triggering abuse protection
- **THEN** the system persists one postcard and independently retryable delivery work for every unique accepted address before initiating any recipient delivery.

#### Scenario: Visitor starts from the postcard entry point

- **WHEN** a visitor follows a public postcard call to action without a selected work
- **THEN** the system SHALL provide a navigable path to select a published artwork before composition.

#### Scenario: Visitor starts from a published artwork

- **WHEN** a visitor chooses to send a published artwork as a postcard
- **THEN** the system SHALL open the composer with that artwork's identity and a usable path to change the selection.

#### Scenario: JavaScript is unavailable

- **WHEN** a visitor selects an artwork, corrects the form, or submits a valid postcard without JavaScript
- **THEN** the system SHALL complete the same journey through ordinary navigable pages and form responses.

#### Scenario: Visitor formats a postcard message

- **WHEN** a JavaScript-enabled visitor applies supported formatting and submits a message
- **THEN** the browser and server independently sanitise the markup to `p`, `b`, `i`, `ul`, `ol`, and `li` elements without attributes, count the 300-character limit against visible Unicode text rather than markup, and preserve the safe formatting on recipient-page and email rendering.

#### Scenario: Visitor composes without JavaScript

- **WHEN** JavaScript or the rich-text dependency is unavailable
- **THEN** the visitor can enter and submit the same required message through an ordinary labelled textarea without seeing non-functional formatting controls.

#### Scenario: Invalid rich-text submission returns to the form

- **WHEN** a submission containing markup fails any validation or abuse check
- **THEN** the server re-renders only sanitised supported markup, the enhanced editor restores that safe value after an ordinary or HTMX response, and no untrusted element or attribute becomes active DOM.

#### Scenario: Visitor changes the recipient rows

- **WHEN** a visitor adds or removes a recipient row before sending
- **THEN** the server re-renders the compose form with the posted values, never removes the final row, and never permits more than five rows regardless of whether HTMX is available.

#### Scenario: Submitted recipients contain duplicates or trailing blanks

- **WHEN** a visitor sends an ordinary or HTMX form submission containing duplicate addresses or blank optional rows
- **THEN** the server drops blank optional rows, deduplicates normalised addresses in posted order, and reports every remaining invalid address honestly.

#### Scenario: Submitted recipients exceed the server limit

- **WHEN** a hand-crafted ordinary or HTMX submission contains more than five unique valid recipient addresses after normalisation and deduplication
- **THEN** the server re-renders the compose form with a five-recipient-limit error, creates no postcard or delivery work, and does not silently discard a valid address.

### Requirement: Postcard delivery and recipient reading are recoverable

The system SHALL track delivery state independently for every postcard address, make repeated external delivery attempts idempotent per address, and provide every recipient the same real public postcard URL rather than an email-only rendering or one public page per address. Confirmation and sender-status surfaces SHALL list all accepted recipients in masked form. The notification email SHALL show the selected artwork image when available, identify the selected work and sender, render the sanitised postcard message, state the link's expiry, and provide the shared postcard link. It SHALL show period music only when a published period-music work matches the selected work, and SHALL omit unavailable artwork or music details rather than rendering sample content.

#### Scenario: Recipient follows a delivered postcard link

- **WHEN** a recipient opens a valid postcard URL
- **THEN** the application renders the selected work, message, sender context, and any applicable published period-music card derived from the work rather than from a sender preference.

#### Scenario: One recipient delivery needs retry

- **WHEN** delivery to one address fails after another address for the same postcard succeeds
- **THEN** retry state and idempotency remain scoped to the failed address, the successful delivery is not repeated, and both deliveries retain the same postcard URL.

#### Scenario: A recipient delivery cannot be completed

- **WHEN** a postcard delivery reaches a terminal failure after its permitted attempts
- **THEN** the system SHALL record that recipient's failed state without representing the postcard as successfully sent.

#### Scenario: Notification email describes the postcard

- **WHEN** the system renders a recipient notification email
- **THEN** the email SHALL contain the artwork image and details when available, the escaped sanitised message, the expiry context, the shared postcard link, and a period-music panel only when a matching published work exists, and SHALL NOT contain recipient addresses or sender-control material.

## ADDED Requirements

### Requirement: Postcard submission is actionable and single-intent

The system SHALL retain valid entered values and present a specific corrective outcome for all expected submission rejections, including invalid or unavailable CAPTCHA, throttling, unavailable artwork, invalid recipients, and persistence unavailability. One visitor send intent SHALL create no more than one postcard and one delivery work item per selected recipient.

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

The system SHALL distinguish queued work from recipient delivery, identify the selected work and the accepted recipients in masked form, and give the sender a safe route to create another postcard.

#### Scenario: Postcard is accepted for delivery

- **WHEN** the system persists valid postcard work before an external delivery attempt
- **THEN** the confirmation SHALL state that the postcard is queued rather than claiming that it has been sent.

### Requirement: Postcard composition supports accessible form completion

The composer SHALL keep its primary fields and submit action visible and usable at supported responsive widths, keyboard focus SHALL reach the first required field, and validation feedback SHALL be associated with the corrected form state.

#### Scenario: Visitor composes on a narrow or wide viewport

- **WHEN** a visitor opens the composer at a supported responsive viewport
- **THEN** artwork context, all required fields, and the submit action SHALL remain usable without horizontal clipping or a trapped scroll position.
