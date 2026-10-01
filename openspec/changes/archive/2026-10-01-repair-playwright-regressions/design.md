# Design

## Context

`AdmissionLimiter` hard-codes `admissionDraftBudget = 3` and `admissionPublishBudget = 3` and is constructed in the itinerary handler. Deployment settings may only be read through `internal/config`. The Playwright runner and the CI job both start WGA with `WGA_ENV` unset or set to `test`, and every request comes from one loopback client.

## Goals / Non-Goals

**Goals:** a green suite without weakening production limits; fix product regressions at their source.

**Non-Goals:** redesigning client identity or accounting.

## Decisions

- **Configurable budgets instead of restructured specs.** The settings are `WGA_ITINERARY_DRAFT_BUDGET` and `WGA_ITINERARY_PUBLISH_BUDGET`, parsed and validated in `internal/config` and passed to the limiter constructor. The runner and CI set high values explicitly. Alternative considered: restructuring the specs to share at most three drafts per hour. Rejected because fully parallel specs would be coupled to hidden global state and ordering.
- **Explicit settings, not derived from `WGA_ENV=test`.** Relaxing limits implicitly by environment name risks a mislabelled deployment running without limits. An explicit value is visible in configuration and logs.
- **Treat the viewer attribute as a regression.** The overhaul replaced the plate container with an `<a data-viewer>` and dropped `data-viewer-no-navbar`, while `bootstrap.ts` still honours it and the specs still expect it. Restoring it is a one-attribute fix that brings back pre-overhaul behaviour.
- **Update drift assertions only against a source of truth.** Each drift fix names what it follows: the catalogue-exploration spec (no file weight) or `internal/assets/reference/visual-overhaul.html` (logo structure, footer columns). If the reference disagrees with the current markup, the markup is fixed instead of the spec.

## Risks / Trade-offs

- [The no-JavaScript "not stable" cause is still unknown] → The first task diagnoses it from the retained traces. If the cause turns out to be an application animation or layout loop, the fix lands in CSS or templates, not in test timeouts. No `force: true` clicks.
- [The overflow fixes could shift layouts elsewhere] → Re-run the public-shell and reference-page viewport specs alongside the targeted specs.
