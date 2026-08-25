## MODIFIED Requirements

### Requirement: Public image surfaces select authoritative renditions
The system SHALL provide handlers with finished PocketBase image URLs resolved against only these visual-overhaul `Wx0` delivery profiles: 120, 200, 400, 500, 600, 700, 800, 900, 1000, 1100, 1400, 1600, and 2000. It SHALL use 120 for itinerary tray chips; 200 for search-panel and picker rows; 400 for related-work and timeline cards; 500 for work cards, artist cards, artist-index portraits, and itinerary-editor plates; 600 for artist-record portraits and unsized work fallback; 700 for postcards and small Dual Mode plates; 800 for guided-tour cards; 900 for work of the day and cookie notice; 1000 for tour title plates; 1100 and 1600 for medium and large Dual Mode plates; 1400 for artwork records and tour picture pages; and 2000 only for deliberate artwork viewing. A handler SHALL request the assigned thumbnail only when its target width is smaller than the source width; otherwise, or when source dimensions are unavailable, it SHALL use the original URL without a thumbnail query so PocketBase does not generate an upscale.

#### Scenario: Handler prepares an image surface
- **WHEN** a handler renders a public artwork or portrait surface
- **THEN** it resolves the assigned delivery profile against the source width and passes either the eligible thumbnail URL or the original URL to the template without the template appending a thumbnail query parameter.

#### Scenario: Visitor opens a deliberate image viewer
- **WHEN** a visitor opens the image viewer from an artwork record, tour picture, itinerary slideshow, or supported Dual Mode plate
- **THEN** the viewer requests the 2000 rendition only when the source is wider than 2000 pixels and otherwise receives the original without upscaling.

#### Scenario: Visitor browses catalogue results
- **WHEN** a visitor renders catalogue results in grid or list presentation
- **THEN** grid cards resolve against the `500x0` profile and list rows resolve against the `200x0` profile, using the original for any source that is not wider than the assigned target.

#### Scenario: Visitor opens an artwork record
- **WHEN** a visitor renders an artwork record with an available image
- **THEN** the record plate resolves against the `1400x0` profile and the deliberate viewer resolves against the `2000x0` profile, using the original wherever the source is not wider than the assigned target.

#### Scenario: Visitor views an artist portrait
- **WHEN** an artist with a portrait appears in the artist index or on their record page
- **THEN** the index resolves against the `500x0` profile and the record resolves against the `600x0` profile, using the original wherever the source is not wider than the assigned target.

### Requirement: Image URL construction is handler-owned
The system SHALL construct public PocketBase image URLs through the URL utility using named delivery profiles and source dimensions. Templates SHALL receive complete thumbnail-or-original URLs and SHALL NOT construct thumbnail query parameters.

#### Scenario: Handler populates a card image
- **WHEN** a handler prepares an image for a public card surface
- **THEN** it selects the named card rendition through the URL utility before rendering the template.

### Requirement: Browsing cards do not invoke the image viewer
The system SHALL treat artwork grid and list images as navigational card content rather than image-viewer galleries. The artwork record SHALL remain the deliberate viewer entry point.

#### Scenario: Visitor selects an artwork grid image
- **WHEN** a visitor activates an image in an artwork grid
- **THEN** the interaction navigates to the artwork record and does not open the image viewer for the grid rendition.

### Requirement: Missing images retain the existing fallback
The system SHALL retain the existing labelled or static missing-image fallback when an artwork or portrait filename is absent, and SHALL NOT generate a thumbnail URL for an absent filename.

#### Scenario: Artwork has no image filename
- **WHEN** a handler renders an artwork with no image filename
- **THEN** the existing no-image fallback is rendered without a PocketBase thumbnail request.
