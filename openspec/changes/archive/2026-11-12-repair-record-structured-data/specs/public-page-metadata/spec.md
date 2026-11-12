## ADDED Requirements

### Requirement: Record pages describe their visible record

Artist, artwork and artist-index pages SHALL describe their records in one JSON-LD block whose `@context` is `https://schema.org` and whose `@graph` holds typed nodes. Each node SHALL contain only values the page visibly renders, SHALL omit a property whose value is missing rather than emitting an empty value or object, and SHALL NOT repeat `@context` on nested nodes. The serialised block SHALL contain no literal `<` character.

- An artwork page SHALL describe the artwork as a `VisualArtwork` with `@id` `{canonical URL}#primary`, `url` equal to the page's canonical URL, `name` equal to the artwork title, and, when shown, `image` as the absolute reproduction URL, `artMedium` as the technique, `artform` as the art type, `dateCreated` as a four-digit creation year, and `description` as the plain-text commentary. It SHALL reference its artist through `creator` with the artist record's `@id`, `name` and `url`.
- An artist page SHALL describe the artist as a `Person` with `@id` `{canonical artist URL}#primary` and `url` equal to the canonical artist URL, with its filing name, life years, places, profession, portrait and plain-text biography when shown. Paragraph boundaries in the biography SHALL be separated by a space.
- An artist-index page SHALL describe each listed artist as a `Person` with only `@id`, `name`, `url`, `birthDate` and `deathDate`.

#### Scenario: Artwork record page

- **WHEN** a client requests a canonical artwork page
- **THEN** its record JSON-LD holds a `VisualArtwork` whose `url` is the page's canonical URL and whose `creator` references the artist record's `@id`

#### Scenario: Missing values

- **WHEN** an artist has no recorded birthplace, profession or portrait
- **THEN** the `Person` node omits `birthPlace`, `hasOccupation` and `image` and contains no empty object

#### Scenario: Artist index

- **WHEN** a client requests the artist index
- **THEN** each listed artist's `Person` node contains no biography, place, profession or image

### Requirement: Visible breadcrumbs are described

Artist, artwork and selection pages that render a breadcrumb trail SHALL include a `BreadcrumbList` with `@id` `{canonical URL}#breadcrumb` in their JSON-LD. Its items SHALL follow the visible linked steps in order, each with `position`, `name` and an absolute `item` URL, ending with the current page's canonical URL. A trail with fewer than two steps SHALL NOT produce a `BreadcrumbList`.

#### Scenario: Artwork breadcrumb

- **WHEN** a client requests a canonical artwork page
- **THEN** its `BreadcrumbList` lists the artist index, the artist record and the artwork record in that order

#### Scenario: Selection breadcrumb

- **WHEN** a client requests a published selection page
- **THEN** its `BreadcrumbList` lists the artist index, the artist record and the selection's canonical URL
