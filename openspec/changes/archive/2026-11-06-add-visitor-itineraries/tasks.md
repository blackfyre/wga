## 1. Persistence and ownership

- [x] 1.1 Define itinerary, stop, and publish-state collections, indexes, limits, and expiry migration tests.
  - Evidence: `internal/migrations/1787097600_add_itineraries.go` defines both collections, lifecycle fields, unique draft/token/stop indexes and the expiry index. `go test ./internal/migrations -cover` passes, including `TestItinerariesMigrationCreatesCollectionsAndIndexes` and uniqueness/rollback tests; `TestPublishTransitionsAndConsumesDraft` verifies the fixed one-year expiry.
- [x] 1.2 Implement session draft identification and feature-owned draft workflow contracts.
  - Evidence: `internal/itineraries/session.go` and `workflow.go` own capability-token identification and draft workflows. `go test ./internal/itineraries ./internal/handlers/itineraries -cover` passes, including token validation, one-draft ownership, isolation, CSRF and origin checks. The stale signed-cookie design wording was corrected with the coordinator's approval; the established security implementation is unchanged.

## 2. Draft interaction

- [x] 2.1 Add itinerary-tray and add controls to supported cards, artwork records, and Dual Mode panes with progressive-enhancement routes.
  - Evidence: `internal/assets/templ/components/itinerary_tray.templ`, the shared add controls and `internal/handlers/itineraries/draft.go` provide the interaction. The itinerary, itinerary-task92 and itinerary-session Playwright specs pass on port 8104, exercising cards, Dual Mode, reload, tray clear, session fragments and ordinary form confirmation.
- [x] 2.2 Implement add, remove, reorder, and narration mutations with validation and HTMX fragments.
  - Evidence: `internal/itineraries/workflow.go` and `internal/handlers/itineraries/draft.go` implement the mutations. Passing Go tests cover duplicate/limit enforcement, ordering, sanitisation, ownership and HTMX/OOB targets; the browser publication scenario exercises arranging, removal, clear and narration.

## 3. Publication and viewing

- [x] 3.1 Implement rate-limited publishing, moderation state, share tokens, and expiry.
  - Evidence: `internal/itineraries/workflow.go`, `admission.go`, `hooks.go` and `internal/handlers/itineraries/publish.go` implement immediate approved publication, operational rejection, immutable random tokens and one-year expiry. Passing workflow/hook/HTTP tests verify rate limits, concurrency, immutability, rejection, public 410 responses and publisher-session privacy.
- [x] 3.2 Render builder, publish confirmation, public itinerary, and one-stop-at-a-time slideshow pages with server fallback navigation.
  - Evidence: itinerary pages in `internal/assets/templ/pages/` and `internal/handlers/itineraries/view.go` render the flow. All 21 relevant browser scenarios pass together on port 8104, including viewer geometry, keyboard/HTMX transitions, publication receipts and a cookie-less recipient following next/previous links and reloading with JavaScript disabled. This closes the earlier partial verification note.
- [x] 3.3 Add scheduled purge and recovery-safe tests.
  - Evidence: `internal/crontab/main.go` registers `internal/itineraries/purge.go` daily. `go test ./internal/itineraries -run 'TestPurge|TestRegisteredPurge' -count=1` passes: expiry, abandoned drafts, cascade deletion, live retention, idempotency, the registered callback, transaction rollback after a later deletion fails, and a successful retry.

## 4. Verification

- [x] 4.1 Add focused workflow, security, and browser coverage for drafts, sharing, expiry, and no-JavaScript navigation.
  - Evidence: passing workflow, migration and HTTP tests cover drafts, bounded/sanitised narration, CSRF, origin/session isolation, API denial, public sharing, moderation and expiry. New tests assert publisher secrets never appear in a cookie-less public response and purge recovery is atomic; the 21 passing Playwright scenarios include ordinary publication forms and ordinary multi-stop recipient navigation without JavaScript. No staging or production access was needed.
