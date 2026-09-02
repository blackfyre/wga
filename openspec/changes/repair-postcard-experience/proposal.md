## Why

The postcard workflow is currently unusable at normal desktop widths, can fail without explaining why, and confirms only that work was queued rather than whether a recipient was reached. The public entry point also cannot lead a visitor into a usable artwork-selection and composition journey.

This change restores a dependable, accessible postcard experience while preserving the existing privacy model, persisted-before-send delivery workflow, and recipient bearer-link boundary.

## What Changes

- Replace the constrained postcard dialog with a dedicated, bookmarkable composer page that works with and without JavaScript.
- Make the public postcard entry point lead visitors to artwork selection, and preserve the chosen published artwork through composition and return navigation.
- Make submission states recoverable: validate and render actionable form feedback for every expected rejection, prevent duplicate creation from a single send intent, and retain entered values on correction.
- Support the existing bounded multi-recipient delivery model in the composer, including clear recipient validation and no silent dropping of submitted addresses.
- Make confirmation and delivery email copy accurately describe the selected work, message, music availability, expiry, and delivery state without disclosing recipient addresses or bearer tokens unnecessarily.
- Add browser and workflow coverage for the compose, rejection, duplicate-submit, delivery-state, and no-JavaScript journeys.

## Capabilities

### Modified Capabilities
- `postcard-sharing`: composition, selection, validation, confirmation, delivery communication, and recipient handling requirements for postcard sharing.

## Impact

- Affects postcard handlers and workflow, delivery state and migrations, the postcard templates and artwork/home entry points, client dialog/HTMX behaviour, email rendering, and postcard browser/workflow tests.
- Retains the existing recipient-token confidentiality boundary and uses forward-only schema changes for submission idempotency and delivery-state reconciliation.
- Does not add visitor accounts, a campaign/bulk-email system, a new mail provider, public postcard browsing, or a replacement administration interface.
