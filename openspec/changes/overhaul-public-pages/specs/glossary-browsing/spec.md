## ADDED Requirements

### Requirement: Public glossary route

The system SHALL expose a public glossary page backed by the existing Glossary collection and render each persisted expression with its definition.

#### Scenario: Visitor opens the glossary

- **WHEN** a visitor navigates to the public glossary route
- **THEN** the system renders glossary terms from the Glossary collection in the public page layout.

### Requirement: Alphabetical glossary browsing

The system SHALL provide an accessible A–Z index that filters the glossary to terms beginning with the selected letter.

#### Scenario: Visitor selects a letter

- **WHEN** a visitor activates a letter from the glossary index
- **THEN** the glossary renders only terms beginning with that letter and identifies the selected letter as current.

### Requirement: Glossary text search

The system SHALL let visitors search persisted expressions and definitions and SHALL provide an empty state and a reset path.

#### Scenario: Visitor searches glossary text

- **WHEN** a visitor enters a query in the glossary search field
- **THEN** matching terms and definitions render in the glossary result block.

#### Scenario: No glossary terms match

- **WHEN** a glossary query produces no matches
- **THEN** the page displays the reference empty state and a control that clears the query.
