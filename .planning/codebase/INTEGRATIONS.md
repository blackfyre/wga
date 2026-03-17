# Integrations

## Core Platform Integration

- PocketBase is the central integration point for persistence, routing hooks, admin auth, cron, file storage settings, and email delivery.
- Collection setup and app configuration are applied through migrations in `internal/migrations/`.
- Query access uses PocketBase APIs and direct SQL against the PocketBase-managed database, for example in `internal/repositories/landing.go`.

## Object Storage

- S3-compatible file storage is configured in `internal/migrations/1687801090_initial_settings.go`.
- Required env vars include `WGA_S3_ENDPOINT`, `WGA_S3_ACCESS_KEY`, `WGA_S3_ACCESS_SECRET`, `WGA_S3_BUCKET`, and `WGA_S3_REGION`.
- Local development uses MinIO via `devenv.nix`.
- File URLs are assembled in helpers such as `internal/utils/url.go` and `internal/utils/url/main.go`.

## Email Delivery

- Postcard delivery uses PocketBase mailer integration in `internal/crontab/postcard.go`.
- Sender metadata comes from `WGA_SENDER_NAME` and `WGA_SENDER_ADDRESS`.
- SMTP is enabled/configured in `internal/migrations/1687801090_initial_settings.go` via `WGA_SMTP_HOST`, `WGA_SMTP_PORT`, `WGA_SMTP_USERNAME`, and `WGA_SMTP_PASSWORD`.
- Email HTML is rendered from `internal/assets/views/emails/postcard.html` through `internal/assets`.

## CAPTCHA

- Google reCAPTCHA verification is implemented in `internal/handlers/postcards/recaptcha.go`.
- Verification posts to `https://www.google.com/recaptcha/api/siteverify`.
- The feature is gated by `WGA_RECAPTCHA_SECRET` in `internal/handlers/postcards/save.go`.
- Local/dev flows intentionally skip provider verification when the secret is unset.

## External HTTP Calls

- `internal/repositories/contributors.go` issues outbound HTTP requests to a configured contributor API endpoint.
- Playwright postcard tests read `MAILPIT_URL` to inspect the local mail UI endpoint for delivered email messages.
- No webhook receivers were identified in the current codebase.

## Local Service Dependencies

- The `mailhog` service is enabled in `devenv.nix` for local email capture, while `MAILPIT_URL` points Playwright at the browser UI used to inspect those captured messages.
- MinIO is enabled in `devenv.nix` for local object storage emulation.
- Environment-derived base URL settings use `WGA_PROTOCOL` and `WGA_HOSTNAME`.

## Scheduled Work

- Cron jobs are registered from `internal/crontab/main.go`.
- `internal/crontab/postcard.go` polls queued postcard records and sends email notifications.
- `internal/crontab/sitemap.go` generates sitemap artifacts on a schedule.

## Security-Sensitive Configuration Surface

- Admin bootstrap uses `WGA_ADMIN_EMAIL` and `WGA_ADMIN_PASSWORD` in `internal/migrations/1695700169_default_admin.go`.
- App URL and sender settings are part of migration-time setup, so environment correctness directly affects runtime behavior.
- Generated codebase docs should avoid recording concrete secret values; only variable names belong here.
