# Glossary Browsing

## Purpose

Define the public glossary route, alphabetical browsing, and text search behaviour.

## Requirements

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

### Requirement: Defined terms expose an in-prose glossary definition
The system SHALL render a persisted glossary term in running prose as a keyboard-reachable dotted-underlined control whose accessible name and hover/focus surface contain the term's definition.

#### Scenario: Reader focuses a defined term
- **WHEN** a keyboard visitor focuses a glossary term in a biography, commentary, or other supported prose
- **THEN** its definition becomes available without navigating away from the sentence.

### Requirement: Shared help tips explain interface terms in place
The system SHALL render a keyboard-reachable help-tip marker for supported interface explanations and keep its inverted tip usable near the top of a scrolling surface.

#### Scenario: Visitor focuses a help tip
- **WHEN** a visitor focuses a help-tip marker
- **THEN** the explanatory text is exposed as the marker's accessible name and visible tip.
