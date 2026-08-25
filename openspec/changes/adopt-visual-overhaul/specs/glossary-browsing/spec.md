## ADDED Requirements

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
