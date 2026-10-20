# Proposal

## Why

A distributed client flooded `/dual-mode` from rotating residential addresses at about 100,000 requests per hour. Each request, including those rejected in a millisecond by admission, wrote a PocketBase request activity log and each rejection wrote a second protection log. With 30-day retention the auxiliary log database reached 8.95 GB within six days, its page cache pushed the container towards its 10 GB memory limit, and every cheap rejection still cost a database write.

## What Changes

- Persist PocketBase request activity logs only for failed requests: successful requests and expected admission rejections no longer write a request log row, while errors and other 5xx responses still do. Other application logs, such as scheduled-run outcomes, keep their current levels and persistence.
- Retain persisted logs for 7 days instead of 30.
- Emit protection decision and cancellation events at most once per event, route profile, and interval, with the number of suppressed events. Admission metrics remain exact.

## Capabilities

### Modified Capabilities

- `public-request-protection`: protection events are bounded per interval, and public request volume does not translate into one persisted log write per successful or rejected request.

## Impact

Router middleware and protection event logging in `internal/handlers`, a settings migration in `internal/migrations/`, and their tests. Request-level telemetry for successful requests remains in OpenTelemetry and Sentry. Applying the retention change deletes persisted logs older than 7 days once at startup; the deployed data directory is currently recreated on each deployment, so no large existing log set is expected. The auxiliary database file does not shrink without a manual `VACUUM`.
