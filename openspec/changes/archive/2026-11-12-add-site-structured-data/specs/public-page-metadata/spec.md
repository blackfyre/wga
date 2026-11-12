## ADDED Requirements

### Requirement: Site-level structured data on every public page

Every full public HTML document rendered with the shared public document head SHALL contain exactly one `<script type="application/ld+json">` block holding a site-level `@graph` with `@context` `https://schema.org`. The graph SHALL contain:

- a `WebSite` node with `@id` `{public root}/#website`, `name` `Web Gallery of Art`, `url` equal to the public root URL, `inLanguage` `en`, and `publisher` referencing the `Organization` node by `@id`;
- an `Organization` node with `@id` `{public root}/#organization`, `name` `Web Gallery of Art`, and `url` equal to the public root URL.

The public root URL SHALL derive from the configured public URL and end with `/`. The serialised block SHALL contain no literal `<` character. Record-level JSON-LD that a page already emits SHALL remain unchanged.

#### Scenario: Visitor loads a public page

- **WHEN** a client requests a full public HTML document, such as `/` or `/glossary`
- **THEN** its `<head>` contains one site-level JSON-LD block whose `@graph` holds the `WebSite` and `Organization` nodes on the configured public root URL

#### Scenario: Record page keeps its record structured data

- **WHEN** a client requests a canonical artwork page
- **THEN** the page contains the site-level graph and its existing `VisualArtwork` JSON-LD

### Requirement: Open Graph locale

Every full public HTML document SHALL emit `<meta property="og:locale" content="en_GB">`.

#### Scenario: Link-preview scraper reads a page

- **WHEN** a client requests any full public HTML document
- **THEN** the Open Graph tags include `og:locale` with the value `en_GB`
