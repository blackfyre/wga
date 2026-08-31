## Purpose

Publish reliable, discoverable XML sitemaps for the Web Gallery of Art's canonical public collection pages.

## ADDED Requirements

### Requirement: Canonical sitemap index publication
The system SHALL publish a valid XML sitemap index at `/sitemap.xml`. The index SHALL reference only same-origin child sitemap URLs beneath `/sitemap/`, and every referenced child sitemap SHALL be retrievable as valid XML.

#### Scenario: Crawler retrieves the sitemap index
- **WHEN** a crawler requests `/sitemap.xml`
- **THEN** it receives a successful XML sitemap index response that references the published child sitemap URLs

#### Scenario: Crawler follows a child sitemap reference
- **WHEN** a crawler requests a child sitemap URL named by the index
- **THEN** it receives the corresponding successful XML URL-set response

### Requirement: Canonical public collection URL inclusion
The system SHALL include the current canonical URLs of published artists and published artworks in its child sitemaps. It SHALL exclude unpublished records and records that cannot produce a valid canonical public URL.

#### Scenario: Published collection records are included
- **WHEN** sitemap generation runs with published artists and artworks
- **THEN** the generated child sitemaps contain their canonical public URLs

#### Scenario: Non-public or incomplete records are excluded
- **WHEN** sitemap generation encounters an unpublished record or an artwork without a resolvable public artist URL
- **THEN** that record is absent from the published sitemap set and the generation outcome records the exclusion

### Requirement: Durable, complete sitemap publication
The system SHALL store the generated sitemap set beneath the configured PocketBase data directory. It SHALL make a newly generated set public only after its index and all referenced child maps have been written successfully, preserving the last complete set when a later generation fails.

#### Scenario: Successful regeneration replaces the previous set
- **WHEN** sitemap regeneration completes successfully
- **THEN** subsequent sitemap requests expose only the complete newly generated set

#### Scenario: Regeneration fails
- **WHEN** sitemap regeneration fails after a previous complete set exists
- **THEN** the previous complete set remains available and the application records the failure without terminating

### Requirement: Sitemap lifecycle and observability
The system SHALL attempt sitemap generation after the application is ready and SHALL regenerate the sitemap daily. It SHALL record the outcome, including completion or failure and the number of published URLs, in the application logs.

#### Scenario: Application starts with no prior sitemap
- **WHEN** the application becomes ready with no generated sitemap set
- **THEN** it attempts to generate and publish a complete sitemap set before relying on the daily schedule

#### Scenario: Scheduled generation fails
- **WHEN** the daily sitemap generation encounters an error
- **THEN** the application remains running and records a failed generation outcome

### Requirement: Sitemap discovery
The system SHALL publish `robots.txt` with a `Sitemap:` directive naming the absolute canonical URL of `/sitemap.xml`.

#### Scenario: Crawler discovers the sitemap
- **WHEN** a crawler requests `/robots.txt`
- **THEN** the response contains a `Sitemap:` directive for the canonical sitemap index URL

### Requirement: Browser-readable sitemap presentation
The sitemap index and child sitemap XML responses SHALL reference a same-origin XSL stylesheet. The stylesheet SHALL render a human-readable browser view using the site's current visual styling without changing the XML sitemap data or requiring JavaScript.

#### Scenario: Browser renders a sitemap
- **WHEN** a browser opens the sitemap index or a child sitemap
- **THEN** it renders a readable styled sitemap view using the site's visual language

#### Scenario: Stylesheet cannot load
- **WHEN** the sitemap XSL stylesheet is unavailable
- **THEN** the sitemap response remains valid XML that a crawler can parse
