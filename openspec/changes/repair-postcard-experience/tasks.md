## 1. Durable postcard state

- [x] 1.3 Implement single-intent postcard submission and separate successful-send versus rejected-attempt admission; recover only the existing postcard outcome for an exact retry and clear its submission key at expiry; verify concurrent/retry, CAPTCHA rejection, throttle, and no-duplicate recipient-delivery tests pass.
- [x] 1.4 Implement recipient-state projection, terminal parent reconciliation, and immutable transport history across retries; verify mixed terminal outcomes and worker-race tests preserve delivery truthfulness.

## 2. Composer and navigation journey

- [x] 2.1 Make the existing selected-artwork postcard route render a full page or `#mc-area` fragment as appropriate, and connect postcard landing, artwork selection, and return navigation; verify focused handler/template tests cover ordinary and HTMX responses for published, missing, and changed artwork selections.
- [x] 2.2 Replace postcard dialog composition with an accessible full-page server-rendered composer containing one to five recipient controls, selected-work context, and truthful music availability; run `templ generate` and verify focused template tests plus responsive browser checks at 390px, 834px, and 1440px.
- [ ] 2.3 Render actionable expected rejections in the owning composer response while preserving non-sensitive values, and add idempotent progressive client enhancement only where needed for recipient rows and in-flight submission state; verify JavaScript-disabled form correction and HTMX validation/CAPTCHA/throttle browser journeys, then run `bun run build` and the scoped Biome check.

## 3. Delivery communication

- [ ] 3.2 Update postcard confirmation and recipient notification email content to distinguish queued from sent and include authorised artwork image/details, message, opted-in available music, and expiry context. Port the approved source into `resources/mjml/postcard_notification.mjml` and compile it into the embedded email template; omit unavailable artwork and music fields rather than rendering sample content or an unselected music panel. Verify mail-rendering and delivery tests assert the artwork image, escaped content, sender music opt-in, and recipient bearer confidentiality.
- [x] 3.3 Wire terminal-delivery outcomes into the existing cron workflow and participation lifecycle; verify a fresh Mailpit-backed run delivers once and terminal failures are never reported as a successful send. (Mailpit/cron verification accepted by user.)

## 4. Integrated verification

- [ ] 4.1 Add Playwright coverage for public entry, artwork selection, full-page composition, multi-recipient submission, duplicate submit, error recovery, no-JavaScript navigation, and responsive keyboard operation; run the focused postcard browser suite against a freshly built application with Mailpit.
- [ ] 4.2 Run `templ generate`, `bun run build`, focused postcard Go tests, `go vet ./...`, and `go test ./... -cover`; record any environment-dependent Mailpit or browser limitation rather than claiming unrun coverage.
- [ ] 4.3 Perform an independent review of migration safety, delivery-state handling, recipient-token confidentiality, and the page/fragment interaction contract; address valid findings and rerun affected deterministic checks.
