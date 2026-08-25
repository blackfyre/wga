## ADDED Requirements

### Requirement: Visitor can send a work as a postcard
The system SHALL let an unauthenticated visitor compose a postcard for a published artwork with recipient details, a message, sender details, validation, abuse protection, and an optional period-music inclusion setting.

#### Scenario: Visitor submits a valid postcard
- **WHEN** a visitor submits a valid postcard form without triggering abuse protection
- **THEN** the system persists the postcard and its delivery work before initiating recipient delivery.

### Requirement: Postcard delivery and recipient reading are recoverable
The system SHALL track postcard delivery state, make repeated external delivery attempts idempotent, and provide the recipient a real public postcard URL rather than an email-only rendering.

#### Scenario: Recipient follows a delivered postcard link
- **WHEN** a recipient opens a valid postcard URL
- **THEN** the application renders the selected work, message, sender context, and any opted-in period-music card as a public page.

### Requirement: Recipient bearer tokens remain confidential at rest
The system SHALL retain only a lookup hash in ordinary postcard data and encrypt any recoverable send value with the configured versioned keyring.

#### Scenario: Delivery is queued for retry
- **WHEN** postcard delivery work is persisted before an external attempt
- **THEN** no plaintext recipient bearer token is stored, while an authorised worker can recover the same token for stable retries.
