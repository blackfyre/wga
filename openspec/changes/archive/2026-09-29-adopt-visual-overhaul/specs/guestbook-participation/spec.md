## ADDED Requirements

### Requirement: Visitors can submit a moderated guestbook entry
The system SHALL provide a public guestbook form with name, location, message, honeypot protection, validation, and a moderation state that prevents unreviewed entries from appearing publicly.

#### Scenario: Visitor signs the guestbook
- **WHEN** a visitor submits a valid entry without completing the honeypot
- **THEN** the application records it for moderation and confirms receipt without publishing it immediately.

### Requirement: Published guestbook entries are searchable historical records
The system SHALL render approved entries newest-first with name, location, and date, plus text search and year navigation.

#### Scenario: Visitor filters the guestbook archive
- **WHEN** a visitor searches text or chooses a year
- **THEN** the page shows matching approved entries and preserves the selected state in a usable URL.

### Requirement: Guestbook retention follows explicit moderation outcomes
The system SHALL disclose that an approved entry becomes a public archival record while approval remains in force. PocketBase superusers SHALL own moderation and retention decisions. Withdrawing approval SHALL immediately remove the entry from public queries and irreversibly redact its name, location, and message; unreviewed and rejected private entries SHALL be removed after ninety days.

#### Scenario: Moderator withdraws an approved entry
- **WHEN** a moderator withdraws approval from a published guestbook entry
- **THEN** the entry is no longer publicly reachable or searchable and its visitor-supplied personal fields cannot be recovered from the application record.
