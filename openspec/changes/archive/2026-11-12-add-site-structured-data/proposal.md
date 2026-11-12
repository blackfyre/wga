## Why

Artist, artwork and artist-index pages already describe their records with `Person` and `VisualArtwork` JSON-LD, but no page identifies the site or its publisher, and every other public page carries no structured data at all. Open Graph tags also omit `og:locale`. Search engines, link-preview scrapers and AI citation systems therefore cannot attribute pages to the Web Gallery of Art as a site, and previews fall back to guessing the language.

## What Changes

- Every full public HTML document emits one site-level JSON-LD graph with a `WebSite` node and its publisher `Organization` node, built from the configured public URL.
- Every full public HTML document emits `og:locale` with the value `en_GB`.
- Existing record-level `Person` and `VisualArtwork` JSON-LD stays unchanged.

## Capabilities

### New Capabilities

- `public-page-metadata`: Site-wide search and sharing metadata that every full public HTML document carries.

### Modified Capabilities

- None.

## Non-goals

- `BreadcrumbList` JSON-LD for the artist, artwork and selection breadcrumbs.
- An `Organization.logo`, `sameAs` links or a sitelinks search action.
- Changes to record-level JSON-LD, Twitter tags, `robots.txt` or crawler policy.

## Impact

- The shared public layout's `<head>` and the template metadata helpers.
- No new dependency, configuration or migration.
