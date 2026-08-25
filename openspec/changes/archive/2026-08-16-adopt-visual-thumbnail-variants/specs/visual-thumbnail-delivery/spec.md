## ADDED Requirements

### Requirement: Public image surfaces select authoritative renditions
The system SHALL provide handlers with finished PocketBase thumbnail URLs using the visual-overhaul `Wx0` rendition assigned to the rendered surface. Artwork cards, catalogue grids, artist works, random inspiration, and home recent works SHALL use `500x0`; catalogue list rows SHALL use `200x0`; related artwork cards SHALL use `400x0`; the work of the day SHALL use `900x0`; postcard previews and received postcards SHALL use `700x0`; artwork records SHALL display `1400x0`; the deliberate artwork viewer SHALL use `2000x0`; artist-index portraits SHALL use `500x0`; and artist-record portraits SHALL use `600x0`.

#### Scenario: Visitor browses catalogue results
- **WHEN** a visitor renders catalogue results in grid or list presentation
- **THEN** grid cards receive `500x0` artwork URLs and list rows receive `200x0` artwork URLs.

#### Scenario: Visitor opens an artwork record
- **WHEN** a visitor renders an artwork record with an available image
- **THEN** the record plate receives a `1400x0` URL and the deliberate viewer receives a `2000x0` URL.

#### Scenario: Visitor views an artist portrait
- **WHEN** an artist with a portrait appears in the artist index or on their record page
- **THEN** the index receives a `500x0` portrait URL and the record receives a `600x0` portrait URL.

### Requirement: Image URL construction is handler-owned
The system SHALL construct public PocketBase image URLs through the URL utility using named rendition values. Templates SHALL receive complete URLs and SHALL NOT construct thumbnail query parameters.

#### Scenario: Handler populates a card image
- **WHEN** a handler prepares an image for a public card surface
- **THEN** it selects the named card rendition through the URL utility before rendering the template.

### Requirement: Browsing cards do not invoke the image viewer
The system SHALL treat artwork grid and list images as navigational card content rather than ViewerJS galleries. The artwork record SHALL remain the deliberate viewer entry point.

#### Scenario: Visitor selects an artwork grid image
- **WHEN** a visitor activates an image in an artwork grid
- **THEN** the interaction navigates to the artwork record and does not open ViewerJS for the grid rendition.

### Requirement: Missing images retain the existing fallback
The system SHALL retain the existing labelled or static missing-image fallback when an artwork or portrait filename is absent, and SHALL NOT generate a thumbnail URL for an absent filename.

#### Scenario: Artwork has no image filename
- **WHEN** a handler renders an artwork with no image filename
- **THEN** the existing no-image fallback is rendered without a PocketBase thumbnail request.
