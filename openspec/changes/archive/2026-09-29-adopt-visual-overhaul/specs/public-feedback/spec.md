## ADDED Requirements

### Requirement: Floating feedback links to the public project issue tracker
The system SHALL render an accessible floating feedback link on every public route whose destination is `https://github.com/blackfyre/wga/issues?q=sort%3Aupdated-desc+is%3Aissue+state%3Aopen+`.

#### Scenario: Visitor opens feedback
- **WHEN** a visitor activates the floating feedback link from a public page
- **THEN** the browser navigates to the configured public GitHub issue list without opening an in-application feedback form.

### Requirement: Feedback navigation is progressively available
The system SHALL implement the floating feedback control as an ordinary link without requiring HTMX or JavaScript.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor activates feedback with JavaScript unavailable
- **THEN** the browser follows the same public GitHub issue-list destination.
