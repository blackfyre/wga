## MODIFIED Requirements

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

## ADDED Requirements

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
