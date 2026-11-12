# public-page-metadata Specification

## Purpose

Define the site-wide search and sharing metadata that every full public HTML document carries, so search engines, link-preview scrapers and AI citation systems can identify the Web Gallery of Art as the site and publisher of each page.

## Requirements

### Requirement: Site-level structured data on every public page

Every full public HTML document rendered with the shared public document head SHALL contain exactly one `<script type="application/ld+json">` block holding a site-level `@graph` with `@context` `https://schema.org`. The graph SHALL contain:

- a `WebSite` node with `@id` `{public root}/#website`, `name` `Web Gallery of Art`, `url` equal to the public root URL, `inLanguage` `en`, and `publisher` referencing the `Organization` node by `@id`;
- an `Organization` node with `@id` `{public root}/#organization`, `name` `Web Gallery of Art`, `url` equal to the public root URL, and `logo` equal to the absolute URL of `/assets/images/logo.png`.

The public root URL SHALL derive from the configured public URL and end with `/`. The serialised block SHALL contain no literal `<` character. Record-level JSON-LD SHALL be emitted in its own block.

#### Scenario: Visitor loads a public page

- **WHEN** a client requests a full public HTML document, such as `/` or `/glossary`
- **THEN** its `<head>` contains one site-level JSON-LD block whose `@graph` holds the `WebSite` node and the `Organization` node with its logo, on the configured public root URL

#### Scenario: Record page keeps its record structured data

- **WHEN** a client requests a canonical artwork page
- **THEN** the page contains the site-level graph and its `VisualArtwork` JSON-LD

### Requirement: Open Graph locale

Every full public HTML document SHALL emit `<meta property="og:locale" content="en_GB">`.

#### Scenario: Link-preview scraper reads a page

- **WHEN** a client requests any full public HTML document
- **THEN** the Open Graph tags include `og:locale` with the value `en_GB`

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

### Requirement: Selection pages describe their works

A published selection page SHALL describe the selection in its JSON-LD `@graph` as an `ItemList` with `@id` `{canonical URL}#primary`, `url` equal to its canonical URL, `name` equal to its display title, `numberOfItems`, and one `ListItem` per shown work in display order, each with `position`, `name` equal to the work title and `item` equal to the work's absolute canonical URL.

#### Scenario: Selection with works

- **WHEN** a client requests a published selection page
- **THEN** its JSON-LD holds an `ItemList` naming the selection and listing each shown work's canonical URL in order

### Requirement: Page descriptions are readable plain text

The `description`, `og:description` and `twitter:description` values SHALL be plain text: markup removed, HTML entities decoded, whitespace collapsed to single spaces, and a space kept between adjacent paragraphs, list items, headings and line breaks. A description longer than 160 characters SHALL be cut to its first 157 characters followed by `...`, without splitting a character.

#### Scenario: Multi-paragraph source text

- **WHEN** a page's description source is `<p>First paragraph.</p><p>Second &amp; last.</p>`
- **THEN** its description is `First paragraph. Second & last.`

#### Scenario: Long multi-byte description

- **WHEN** a page's description source is longer than 160 characters and contains multi-byte characters
- **THEN** its description is valid UTF-8 of 157 characters followed by `...`
