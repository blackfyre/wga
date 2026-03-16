# Testing

## Current Test Surface

- Backend tests are standard Go `_test.go` files colocated with source packages.
- Browser tests use Playwright under `playwright-tests/`.
- No dedicated load-testing, fuzzing, or benchmark suite was identified.

## Go Test Coverage Areas

- `internal/crontab/postcard_test.go` covers postcard cron helper behavior.
- `internal/handlers/dual/main_test.go` covers URL and pane-path logic for dual mode.
- `internal/handlers/postcards/recaptcha_test.go` covers captcha verification against a rewritten test transport.
- `internal/repositories/contributors_test.go` and `internal/repositories/landing_test.go` cover repository behavior.
- `internal/utils/*.go` has several tests including cache, artist helpers, and URL/current-page helpers.
- `internal/validation/forms_test.go` covers basic validation helpers.
- `internal/assets/templ/utils/combobox_test.go` covers template utility behavior.

## Test Patterns

- Unit tests dominate; most test files exercise helper logic instead of full request lifecycles.
- Table-driven tests are used where branching complexity is higher.
- HTTP behavior is tested with `httptest` where needed, as seen in reCAPTCHA verification tests.
- Assertions rely on Go standard library patterns rather than external assertion helpers, despite `testify` being present indirectly.

## End-to-End Coverage

- `playwright-tests/artists.spec.ts` covers artist listing/detail flows.
- `playwright-tests/artwork-search.spec.ts` covers artwork filtering combinations and reset behavior.
- `playwright-tests/feedback.spec.ts` covers feedback submission UX.
- `playwright-tests/guestbook.spec.ts` covers guestbook message flow.
- `playwright-tests/postcard.spec.ts` covers postcard sending and checks delivered email content through Mailpit/MailHog APIs.

## Tooling and Execution

- Go tests run with `go test ./... -cover`.
- Playwright is configured in `playwright.config.ts`.
- Browser tests use `WGA_PROTOCOL` and `WGA_HOSTNAME` to derive `baseURL`.
- Playwright keeps traces, screenshots, and video on failure.

## Gaps

- There is no visible request-level handler test harness around PocketBase route registration.
- Cron workflows appear only lightly tested relative to their side effects.
- Contributor API integration has outbound HTTP calls and likely deserves broader failure-path coverage.
- No explicit coverage reporting or threshold enforcement is wired into CI from the files reviewed here.

## Practical Guidance

- Prefer colocated `_test.go` files for backend additions.
- Use table-driven cases for parsing, branching, or filter logic.
- Reach for `httptest` when validating external HTTP integrations.
- Extend Playwright when user-facing flows or form behavior change.
