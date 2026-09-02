# Design: repair postcard experience

## Context

The postcard flow must remain a public, account-free experience. Recipient access is represented by an encrypted, recipient-specific bearer token and delivery is queued before any SMTP attempt. The change improves composition, expected-error recovery, delivery truthfulness, and recipient communication without exposing addresses or recipient tokens.

## Decisions

### Keep submission and delivery state durable

The postcard workflow persists recipient delivery work before SMTP. Submission idempotency prevents duplicate recipient deliveries. Each SMTP retry is an immutable delivery-attempt record; parent postcard state is reconciled from recipient outcomes so a terminal failure is never presented as queued or sent.

### Use a page-first, progressively enhanced composer

The selected-artwork postcard route renders the full composer for ordinary navigation and the `#mc-area` fragment for HTMX navigation. It has one required recipient and up to four optional recipient controls. The server remains the validation authority. Expected validation, CAPTCHA, and throttling responses re-render the composer; JavaScript only enhances the recipient rows, in-flight state, and expected HTMX error swaps.

### Make recipient emails truthful and data-backed

`resources/mjml/postcard_notification.mjml`, ported from `wga-visual-overhaul/project/postcard-email.mjml`, is the approved presentation source for the recipient notification. Its compiled, embedded HTML template must use only data authorised for the recipient: selected artwork image and details, the sender name, escaped message, music availability, expiry, and the recipient’s private link. The selected artwork image uses the established hosted delivery profile when available; the template retains a text plate only as its unavailable-image fallback.

The email must omit unavailable artwork or music fields rather than using the MJML sample content. Its music panel requires both the sender’s music opt-in and a matching published work. It must not include recipient addresses, sender-control material, or another recipient’s bearer link.

### Retain the recipient privacy boundary

Recipient tokens remain the only public capability for viewing a postcard. They are encrypted at rest and purged according to the existing delivery lifecycle. Request and run logs redact sensitive values. No sender status or cancellation route is part of this change.

## Component boundaries

| Area | Owner | Responsibility |
| --- | --- | --- |
| Composer | postcard handlers, Templ pages/components, client bootstrap | page/fragment rendering, validation recovery, and accessible progressive enhancement |
| Submission and delivery | postcard workflow | idempotency, recipient state, retries, and terminal-parent reconciliation |
| Recipient communication | postcard delivery workflow and embedded email view | authorised email data, recipient link, and message rendering |

## Verification

1. Focused workflow tests cover idempotent submission, retry history, terminal states, and recipient-token handling.
2. Templ and Playwright checks cover full-page, HTMX, responsive, keyboard, and no-JavaScript composition paths.
3. Mail rendering and Mailpit-backed tests verify the recipient email content, escaping, expiry, and recipient-link confidentiality.
4. `templ generate`, `bun run build`, focused postcard tests, `go vet ./...`, and the wider Go suite run before closure.

## Risks

- An email template can become misleading if runtime facts are unavailable. Omit independently unavailable artwork or music fields rather than substituting sample values.
- SMTP transport may be ambiguous or fail after queueing. Preserve the attempt history and report the terminal state truthfully; do not claim successful send from queue admission alone.
- Recipient links are bearer credentials. Do not render or log them outside the intended recipient email and view flow.
