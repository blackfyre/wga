# Spec Delta

## ADDED Requirements

### Requirement: Catalogue results highlight text search matches

In the grid and list result presentations, the system SHALL wrap the first case-insensitive occurrence of the active text search term in each artwork result's visible title and visible artist name in a `mark` element. The free-text query SHALL apply to both the title and the artist name. The title filter SHALL apply to the title, and when both are present it SHALL take precedence over the free-text query for the title. The artist text filter SHALL apply to the artist name, and when both are present it SHALL take precedence over the free-text query for the artist name, unless an exact artist-record filter is active. All highlighted and surrounding text SHALL be HTML-escaped. Attribute text (such as image alternative text, tooltips and accessible labels) SHALL remain unhighlighted plain text. The visible text of each result SHALL be the same as without highlighting.

#### Scenario: Free-text query matches the title

- **WHEN** a visitor requests `/artworks?q=madonna` and a result's title is "Madonna and Child"
- **THEN** the result's visible title renders "<mark>Madonna</mark> and Child"
- **AND** the image alternative text remains "Madonna and Child"

#### Scenario: Free-text query matches the artist name

- **WHEN** a visitor requests `/artworks?q=giotto` and a result's artist is "GIOTTO di Bondone" with a title that does not contain the term
- **THEN** the result's artist name renders "<mark>GIOTTO</mark> di Bondone" and its title has no `mark` element

#### Scenario: Highlight survives a results fragment swap

- **WHEN** a visitor types a text query and the debounced results fragment replaces the current results
- **THEN** the swapped-in results highlight the new query term

#### Scenario: Exact artist filter does not highlight the artist text filter

- **WHEN** a visitor requests `/artworks?artist_id=<id>&artist=rem`
- **THEN** no artist name is highlighted because of the artist text filter

#### Scenario: Search text is escaped

- **WHEN** a visitor requests `/artworks?q=<b>` and a result's title contains "<b>"
- **THEN** the response contains `<mark>&lt;b&gt;</mark>` and no unescaped markup from the query or title

#### Scenario: No text search

- **WHEN** a visitor filters only by school, form, type or date range
- **THEN** no result contains a `mark` element
