## ADDED Requirements

### Requirement: Visitor can send a work as a postcard

The system SHALL let an unauthenticated visitor compose one postcard for a published artwork with between one and five recipient addresses, a message, sender details, validation, and abuse protection. The system SHALL normalise and deduplicate recipient addresses and SHALL enforce the five-address limit server-side. The compose form SHALL support adding and removing recipient rows through the existing submission endpoint with or without JavaScript, SHALL retain at least one row, and SHALL NOT offer or accept a sender-controlled period-music inclusion setting.

#### Scenario: Visitor submits a valid postcard

- **WHEN** a visitor submits a valid postcard form without triggering abuse protection
- **THEN** the system persists one postcard and independently retryable delivery work for every unique accepted address before initiating any recipient delivery.

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

The system SHALL track delivery state independently for every postcard address, make repeated external delivery attempts idempotent per address, and provide every recipient the same real public postcard URL rather than an email-only rendering or one public page per address. Confirmation and sender-status surfaces SHALL list all accepted recipients in masked form.

#### Scenario: Recipient follows a delivered postcard link

- **WHEN** a recipient opens a valid postcard URL
- **THEN** the application renders the selected work, message, sender context, and any applicable published period-music card derived from the work rather than from a sender preference.

#### Scenario: One recipient delivery needs retry

- **WHEN** delivery to one address fails after another address for the same postcard succeeds
- **THEN** retry state and idempotency remain scoped to the failed address, the successful delivery is not repeated, and both deliveries retain the same postcard URL.

### Requirement: Recipient bearer tokens remain confidential at rest

The system SHALL use one random 256-bit bearer token for the shared postcard URL, retain only its lookup hash in ordinary postcard data, and encrypt the recoverable send value with the configured versioned keyring in locked delivery state until every recipient delivery reaches successful or final resolution.

#### Scenario: Delivery is queued for retry

- **WHEN** postcard delivery work is persisted before an external attempt
- **THEN** no plaintext recipient bearer token is stored, while an authorised worker can recover the same shared token for stable retries to any outstanding address.
