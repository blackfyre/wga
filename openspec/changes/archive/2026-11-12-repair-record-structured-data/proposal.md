## Why

The record-level JSON-LD on catalogue pages is defective. The artwork page's `VisualArtwork` block has no `@context` or `@type`, so consumers ignore it; it has no `name`, its `url` points to an unserved `/artworks/-{id}` path, its image is an untyped object holding another `image` key, and it uses the comics-specific `artist` property. Every `Person` serialises absent places and occupations as empty objects, runs biography paragraphs together, and repeats `@context` on nested nodes. The artist index describes each artist with a full biography the index does not show, and no page describes its visible breadcrumb trail.

## What Changes

- Artwork pages describe the record as a typed `VisualArtwork` on its canonical URL, with its visible title, reproduction, medium, type, four-digit creation year and commentary, and a `creator` reference to the artist record.
- Artist pages describe the artist as a `Person` on its canonical URL, without empty nodes, and with paragraph boundaries kept in the description.
- The artist index describes each listed artist only by what the index shows.
- Artist, artwork and selection pages describe their visible breadcrumb trail as a `BreadcrumbList`.
- Each record page's JSON-LD is one `@graph` whose nodes link by `@id`.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `public-page-metadata`: adds record-level structured data requirements.

## Non-goals

- An `Organization.logo`, `sameAs` links, or structured data for pages other than the artist index, artist, artwork and selection pages.
- Changes to Open Graph or Twitter tags.

## Impact

- The JSON-LD builders and the artist, artwork, index and selection handlers.
- No new dependency, configuration or migration.
