## Context

The reference defines itineraries as session-owned drafts that can be ordered, narrated, published, and viewed publicly. WGA has artwork routes and HTMX patterns but no visitor-owned persistent workflow.

## Goals / Non-Goals

**Goals:**

- Provide recoverable session drafts, bounded public sharing, moderation, expiry, and purge.
- Keep artwork additions and draft mutations server-owned and progressively enhanced.

**Non-Goals:**

- User accounts, collaborative editing, timeline exploration, or period music.

## Decisions

- Persist drafts and published itineraries in PocketBase collections, with a signed anonymous session cookie identifying one draft. This survives page navigation without introducing accounts.
- Persist an ordered stop list with title, artwork relation, and sanitised narration. Mutations use POST routes with CSRF/session validation and return feature-owned HTMX fragments.
- Publish creates an immutable share token and expiry; a scheduled owner job purges expired records and associated drafts.
- Render public views server-side and use a small feature-local browser helper only for slideshow navigation and prefetch.

## Risks / Trade-offs

- [Anonymous publishing can be abused] → rate-limit publish and add moderation status before public discovery.
- [Session drafts can grow unbounded] → cap stops and purge abandoned drafts.
- [Artwork deletion can orphan stops] → retain a display snapshot and skip unavailable artworks.

## Migration Plan

1. Add collections, indexes, and migration tests.
2. Register handlers and purge schedule before exposing navigation.
3. Deploy with empty drafts; rollback by disabling routes while preserving records for forward cleanup.
