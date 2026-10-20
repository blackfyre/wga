# Tasks

## 1. Itinerary session after the document

- [x] 1.1 Add `GET /itineraries/session` rendering the templ tray with `data-csrf`, `data-count`, and `data-added`, returning an empty tray without issuing a cookie when no session exists, and classify it under origin admission.
- [x] 1.2 Remove the router-wide itinerary session middleware; render stateless add controls and an empty tray placeholder that loads the session fragment.
- [x] 1.3 Inject `_csrf` and apply added/full state through one delegated, swap-safe module in `resources/js/itinerary.ts`.
- [x] 1.4 Accept a same-origin first addition without a session cookie and issue the session; keep the token requirement when a session cookie is present; add the no-JavaScript confirmation form.

## 2. Cache policy

- [x] 2.1 Add the explicit public cache-policy helper and opt in the catalogue route families; keep HTMX and remembered-preference responses `private, no-store`.
- [x] 2.2 Add Go tests asserting that opted-in `GET` and `HEAD` responses carry no `Set-Cookie`, CSRF value, or tray content.

## 3. Edge configuration

- [x] 3.1 Document the Cache Rule, its bypass expressions, and the extended rate-limit route families in `docs/bot-protection-runbook.md`, after confirming the Free-plan capabilities.

## 4. Verification

- [x] 4.1 Playwright: first addition from a cookie-less page, state surviving HTMX swaps, and the no-JavaScript addition.
- [x] 4.2 Security review of the first-addition CSRF rule.
- [x] 4.3 Run `go vet ./...`, `go test ./... -cover`, `golangci-lint run`, and `mise run test:playwright`.
