## ADDED Requirements

### Requirement: Portraits use their assigned release surfaces
The system SHALL use the active resolved biography portrait supplied by `wga-src`, preserving its original portrait path and any artwork-match provenance. Artist-card and artist-index portraits SHALL resolve against the 500 delivery profile and artist-record portraits SHALL resolve against the 600 delivery profile, using the original whenever the active source is not wider than the target or its dimensions are unavailable. Labelled fallbacks SHALL remain for missing portraits.

#### Scenario: Visitor opens an artist record with a portrait
- **WHEN** an artist has an available portrait filename
- **THEN** the record renders the source-eligible 600 portrait rendition or the original without upscaling, and its identity metadata remains consistent with the active portrait source and recorded match provenance.
