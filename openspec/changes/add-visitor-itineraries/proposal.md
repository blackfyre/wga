## Why

Visitors can compare and discover individual works but cannot preserve, narrate, or share a personally curated path through the collection. The updated visual reference defines itineraries as a first-class public workflow.

## What Changes

- Add visitor-owned draft itineraries that collect artworks, permit ordering and narration, and can be published as expiring public views.
- Add a persistent itinerary tray and artwork-page add action.
- Add moderation, rate limiting, sanitisation, expiry, and purge behaviour for public itineraries.

## Capabilities

### New Capabilities
- `visitor-itineraries`: Draft, publish, share, view, expire, and purge visitor-curated artwork itineraries.

### Modified Capabilities
- None.

## Impact

- New PocketBase collections, migrations, handler package, routes, session/draft state, scheduled purge work, Templ pages/components, browser helpers, and Playwright/Go coverage.
