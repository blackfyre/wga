## Why

Visitors who prefer bionic reading have no way to apply that presentation aid while browsing the public gallery. The visual-overhaul reference already defines a constrained, reversible interaction that can be adapted without changing server-rendered content.

## What Changes

- Add an optional, off-by-default bionic-reading preference to public pages.
- Provide an enhanced footer switch that persists the visitor's choice in local storage.
- Apply the presentation only to eligible prose, restore the original text when disabled, and process newly swapped HTMX content.
- Keep the preference client-side: no cookie, server state, or accessibility-performance claim is introduced.

## Capabilities

### New Capabilities
- `bionic-reading`: Provide a reversible, client-side reading-presentation preference for eligible public prose.

### Modified Capabilities

None.

## Impact

- Affected browser code: `resources/js/bootstrap.ts` plus a focused bionic-reading module and its tests.
- Affected shared UI: `internal/assets/templ/components/footer.templ`.
- The existing HTMX lifecycle must continue to initialise presentation for replaced content.
- No API, database, dependency, cookie, or server-rendering changes.
