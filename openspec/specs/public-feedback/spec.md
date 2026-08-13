# Public Feedback

## Purpose

Define public feedback categorisation, contextual persistence, guidance, acknowledgement, and recovery behaviour.

## Requirements

### Requirement: Categorised contextual feedback reports

The system SHALL let visitors submit a feedback report with a required category, required message, optional contact details, and the originating page context while retaining honeypot protection.

#### Scenario: Visitor submits a correction report

- **WHEN** a visitor selects the correction category, enters a valid message, and submits the feedback form
- **THEN** the system persists the category, message, optional contact details, and source-page context as one feedback report.

#### Scenario: Honeypot is completed

- **WHEN** a feedback submission contains honeypot input
- **THEN** the system rejects the submission without storing a report and presents the existing generic failure response.

### Requirement: Feedback message guidance

The system SHALL display the selected report category, a category-appropriate message prompt, and a live visible remaining-character count for the reference message limit.

#### Scenario: Visitor changes report category

- **WHEN** a visitor selects a different feedback category
- **THEN** the form updates its message guidance for that category.

#### Scenario: Visitor enters feedback text

- **WHEN** a visitor types in the feedback message field
- **THEN** the visible remaining-character count reflects the characters still permitted.

### Requirement: Feedback acknowledgement and recovery

The system SHALL present an accessible acknowledgement only after a feedback report is accepted and SHALL preserve a usable form with error feedback if persistence fails.

#### Scenario: Feedback report is accepted

- **WHEN** the feedback workflow accepts a valid report
- **THEN** the dialog presents the reference acknowledgement state and the visitor can dismiss it.

#### Scenario: Feedback persistence fails

- **WHEN** a valid feedback report cannot be persisted
- **THEN** the dialog returns a usable report form and the visitor receives an error notification.
