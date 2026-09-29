# Artist Portraits

## Purpose

Define artist portrait presentation and identity metadata for published artist pages.

## Requirements

### Requirement: Artist portrait presentation

The system SHALL render an artist's optional portrait from the artist record's file reference in the reserved portrait panel and SHALL retain a labelled visual fallback when no portrait reference exists.

#### Scenario: Artist has a portrait

- **WHEN** a visitor opens an artist page whose published artist record has a portrait filename
- **THEN** the page renders the corresponding artist file image with the artist name as alternative text.

#### Scenario: Artist has no portrait

- **WHEN** a visitor opens an artist page whose published artist record has no portrait filename
- **THEN** the page retains the labelled portrait fallback and does not render a broken image URL.

### Requirement: Artist portrait identity metadata

The system SHALL use an artist's portrait file URL as that artist page's Open Graph image and `Person.image` JSON-LD value when a portrait is present.

#### Scenario: Artist has identity portrait metadata

- **WHEN** an artist page is rendered for an artist with a portrait filename
- **THEN** its Open Graph image and `Person.image` JSON-LD identify the same portrait file URL rendered on the page.

#### Scenario: Artist has no identity portrait

- **WHEN** an artist page is rendered for an artist without a portrait filename
- **THEN** its `Person` JSON-LD omits `image` and its Open Graph image preserves the existing first-artwork fallback when one is available.

### Requirement: Portraits use their assigned release surfaces
The system SHALL use the active resolved biography portrait supplied by `wga-src`, preserving its original portrait path and any artwork-match provenance. Artist-card and artist-index portraits SHALL resolve against the 500 delivery profile and artist-record portraits SHALL resolve against the 600 delivery profile, using the original whenever the active source is not wider than the target or its dimensions are unavailable. Labelled fallbacks SHALL remain for missing portraits.

#### Scenario: Visitor opens an artist record with a portrait
- **WHEN** an artist has an available portrait filename
- **THEN** the record renders the source-eligible 600 portrait rendition or the original without upscaling, and its identity metadata remains consistent with the active portrait source and recorded match provenance.
