## ADDED Requirements

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
