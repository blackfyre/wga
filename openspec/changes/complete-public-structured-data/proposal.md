## Why

Three gaps remain after the site-level and record-level structured data work. The publisher `Organization` carries no logo although the site has a brand logo. Selection pages describe only their breadcrumbs, not the works they present. Meta descriptions run adjacent paragraphs together, keep HTML entities encoded, and are truncated by bytes, which can split a multi-byte character.

The artist content renderer that carried its own per-artwork JSON-LD is no longer reachable from any route.

## What Changes

- The publisher `Organization` node names the site's brand logo.
- Selection pages describe the selection as an `ItemList` of the works it shows, in order.
- Meta, Open Graph and Twitter descriptions are plain text with paragraph boundaries kept and entities decoded, truncated to whole characters.
- The unreachable artist content renderer and its tests are removed.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `public-page-metadata`: adds the publisher logo, selection structured data and plain-text descriptions.

## Non-goals

- A new logo asset. The existing logo is 318 × 45 pixels, below Google's 112-pixel minimum for logo rich results; the markup stays truthful but may not earn that result.
- `sameAs` links or structured data for other page kinds.

## Impact

- The template metadata helpers, the JSON-LD builders and the selection handler.
- No new dependency, configuration or migration.
