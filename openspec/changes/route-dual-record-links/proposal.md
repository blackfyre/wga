# Proposal

## Why

In Dual Mode, links inside an artist biography or a selection's commentary are rendered as raw HTML with plain `/artists/…` hrefs. Clicking one leaves Dual Mode for the standard record page and ignores the pane's "LINKS OPEN IN" choice, while every other record link in a pane (work cards, index rows) is routed to the selected pane.

## What Changes

- Record links (artist, artwork, and selection routes) inside a Dual Mode artist biography, a selection preview's commentary, and a selection record's commentary are rewritten to Dual Mode URLs that load the target into the pane chosen by that pane's link-routing setting, with the same progressive-enhancement contract as other pane links (ordinary `href` plus the HTMX request, target, select, and swap).
- Links that are not supported pane content (external URLs, other in-application pages) are left unchanged.
- The standard artist and selection record pages are unaffected.

## Non-Goals

- Rewriting legacy WGA paths in stored commentary; that is `wga-src`'s `resolve-selection-commentary-links` change, after which those links become record routes and are covered here automatically.
- Changing the existing rule that `OPEN SELECTION` and sibling-selection actions always update the pane being read.

## Capabilities

### New Capabilities

### Modified Capabilities

- `catalogue-exploration`: adds a requirement that embedded biography and commentary record links honour Dual Mode pane routing.

## Impact

- `internal/handlers/dual/main.go` (biography and commentary HTML preparation) and its tests.
- `playwright-tests/dual-mode.spec.ts` (new journey).
- `docs/features/dual-mode.md` acceptance criteria.
