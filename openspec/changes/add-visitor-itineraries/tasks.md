## 1. Persistence and ownership

- [ ] 1.1 Define itinerary, stop, and publish-state collections, indexes, limits, and expiry migration tests.
- [ ] 1.2 Implement session draft identification and feature-owned draft workflow contracts.

## 2. Draft interaction

- [ ] 2.1 Add itinerary-tray and add controls to supported cards, artwork records, and Dual Mode panes with progressive-enhancement routes.
- [ ] 2.2 Implement add, remove, reorder, and narration mutations with validation and HTMX fragments.

## 3. Publication and viewing

- [ ] 3.1 Implement rate-limited publishing, moderation state, share tokens, and expiry.
- [ ] 3.2 Render builder, publish confirmation, public itinerary, and one-stop-at-a-time slideshow pages with server fallback navigation.
  - Partial verification (2026-08-29): stop navigation now swaps only the opaque `#itinerary-viewer` fragment, preserves the ordinary server-rendered links, prevents the global HTMX View Transition, and does not replay an entry fade. Focused template/handler/Bun tests, frontend build, scoped vet, and the viewer transition browser scenario passed. The parallel itinerary browser file still has independent builder/tray test-isolation failures and remains unclosed.
- [ ] 3.3 Add scheduled purge and recovery-safe tests.

## 4. Verification

- [ ] 4.1 Add focused workflow, security, and browser coverage for drafts, sharing, expiry, and no-JavaScript navigation.
