## Why

Sentry recorded `WGA-BROWSER-5`: an `InvalidStateError` reporting that a transition was aborted by invalid state. The event is recent and needs source-mapped diagnosis before a focused behavioural fix can be selected.

## What Changes

- Identify the browser API and interaction path that aborts the transition.
- Add a narrowly scoped recovery or guard that preserves navigation and dialog behaviour.
- Add regression coverage for the failing interaction once the event is attributed.

## Capabilities

### New Capabilities
- `browser-transition-resilience`: safe recovery from aborted browser transitions.

### Modified Capabilities

None.

## Impact

- Browser bootstrap and potentially HTMX/dialog interaction handling.
- No server API, data, or authentication changes.
