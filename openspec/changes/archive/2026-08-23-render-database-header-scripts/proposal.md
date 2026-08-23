## Why

The production seed already defines a privileged `strings` record named `scripts:header`, but WGA does not consume it. Operators therefore cannot add trusted database-managed header scripts without rebuilding the application.

## What Changes

- Load the optional `scripts:header` content for full HTML document requests and make it available to the shared layout.
- Render non-empty content verbatim in the document `<head>` so trusted script markup can execute.
- Omit the fragment when the record is missing or empty, and keep the page available when the optional lookup fails.
- Treat the record as privileged executable content rather than as user-authored rich text.
- Do not populate or otherwise modify either production record, add public collection access, change the database schema, or add equivalent footer injection.

## Capabilities

### New Capabilities

- `database-managed-head-markup`: Privileged operators can supply optional trusted markup for the shared document head through the existing `scripts:header` record.

### Modified Capabilities

- None.

## Impact

- Affects the shared Templ layout, request-context preparation, PocketBase `strings` lookup, and focused Go/Templ tests.
- Introduces an intentional stored-script execution boundary: any superuser or producer input able to change `scripts:header` can execute browser code on every full WGA page.
- Adds no dependency, route, public API, collection rule, schema, or seed-data change.
