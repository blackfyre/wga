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

### Requirement: Pre-release GitHub feedback intake

Until the private in-app intake is explicitly restored for the first production release, the public feedback entry point SHALL open the repository's GitHub issue template chooser rather than the existing-issues listing.

#### Scenario: Visitor opens the feedback entry point

- **WHEN** a visitor activates the public feedback link
- **THEN** the visitor reaches the repository's new-issue template chooser.

### Requirement: Feedback-aligned public issue forms

The GitHub issue chooser SHALL offer structured forms for general feedback, collection corrections, technical problems, and suggestions using the same category meanings as the in-app feedback workflow.

#### Scenario: Visitor chooses a collection correction

- **WHEN** a visitor selects the collection-correction form
- **THEN** the form requests the affected page, the current information, the proposed correction, and supporting source information.

#### Scenario: Visitor chooses a technical problem

- **WHEN** a visitor selects the technical-problem form
- **THEN** the form requests the affected page, observed and expected behaviour, reproduction steps, and relevant browser or device details.

#### Scenario: Visitor chooses a suggestion

- **WHEN** a visitor selects the suggestion form
- **THEN** the form requests the underlying problem, desired outcome, and optional proposed solution.

#### Scenario: Visitor chooses general feedback

- **WHEN** a visitor selects the general-feedback form
- **THEN** the form requests a concise message and an optional relevant page.

### Requirement: Public submission privacy guidance

Each GitHub feedback form SHALL state that submitted content is public, SHALL tell visitors not to include personal information, and SHALL NOT request private contact details.

#### Scenario: Visitor reviews a GitHub feedback form

- **WHEN** any feedback issue form is displayed
- **THEN** its public-submission guidance is visible before submission and no contact-detail field is presented.
