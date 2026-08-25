## ADDED Requirements

### Requirement: Guided tours are distinct editorial routes
The system SHALL present guided tours as named-editor, revision-aware, permanent editorial works and SHALL distinguish them from anonymous, expiring visitor itineraries.

#### Scenario: Visitor compares tour and itinerary choices
- **WHEN** a visitor opens the Guided Tours index
- **THEN** the page states the author, shape, length, reading mode, and lifetime distinction between tours and itineraries.

### Requirement: Tour reading renders one addressed page at a time
The system SHALL render a tour title, text, picture, index, or sources page at a stable route with tour context, progress, contents, and previous/next navigation.

#### Scenario: Scholar follows a tour page link
- **WHEN** a scholar opens a numbered tour page URL
- **THEN** the application renders that one page, its place in the tour, and its sources or neighbouring pages without flattening the tour into one document.

### Requirement: Legacy tour material remains reachable
The system SHALL show a non-rebuilt tour as an honest original-layout destination rather than presenting a dead link or pretending it has page-by-page content.

#### Scenario: Visitor selects an unreconstructed tour
- **WHEN** a visitor selects a tour with no rebuilt pages
- **THEN** the title page explains its original layout status and offers its available destination.
