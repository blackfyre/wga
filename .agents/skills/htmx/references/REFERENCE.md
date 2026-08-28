# HTMX Reference Index


Use this index to pick the smallest reference file that matches the task.

## Package Guides

| Domain | File | Use For |
|---|---|---|
| Attributes | `references/attributes.md` | All `hx-*` attributes, their accepted values, modifiers, and inheritance behavior |
| Requests | `references/requests.md` | Triggering requests, headers (request/response), parameters, file uploads, CSRF, CORS, caching |
| Swapping | `references/swapping.md` | Swap strategies, targets, extended CSS selectors, OOB swaps, morphing, view transitions, CSS transitions |
| Events & API | `references/events-api.md` | Lifecycle events, JS API methods, configuration options, extensions, debugging, scripting |
| Patterns | `references/patterns.md` | Common UI patterns: active search, infinite scroll, click to load, edit row, modals, lazy loading, tabs, file upload, progress bar |
| Extensions | `references/extensions.md` | Official extensions: WebSockets (`ws`), SSE (`sse`), Idiomorph (`morph`), response targets, head support, preload |
| Gotchas | `references/gotchas.md` | Common pitfalls, silent failures, accessibility, error handling, architecture decisions, version migration |

## Common Task Routing

- Add AJAX behavior to a button/link/form: read `references/attributes.md`
- Configure when a request fires (polling, debounce, keyboard shortcuts): read `references/requests.md`
- Control where the response goes and how it renders: read `references/swapping.md`
- Update multiple parts of the page from one response: read `references/swapping.md` (OOB swaps section)
- Listen for HTMX events or call the JS API: read `references/events-api.md`
- Set up extensions (SSE, WebSockets, Idiomorph, preload, response-targets, head-support): read `references/extensions.md`
- Implement a common UI pattern (search box, infinite scroll, modals): read `references/patterns.md`
- Integrate with third-party JS libraries: read `references/events-api.md` (scripting section)
- Security hardening (CSRF, CSP, XSS prevention): read `references/requests.md` (security section)
- Avoiding common pitfalls and silent failures: read `references/gotchas.md`
- Accessibility best practices for HTMX: read `references/gotchas.md` (accessibility section)
- Deciding if HTMX is the right fit for a project: read `references/gotchas.md`

## Suggested Reading Order

1. Start with this file.
2. Open one domain file for the immediate task.
3. Open additional files only when cross-domain integration is required.

## Sources

All reference material is derived from official htmx documentation and source code.

| Source | URL |
|---|---|
| htmx documentation (v2) | https://htmx.org/docs/ |
| htmx reference (v2) | https://htmx.org/reference/ |
| htmx 4 documentation | https://four.htmx.org/ |
| htmx 4 migration guide | https://four.htmx.org/docs/get-started/migration/ |
| htmx GitHub repo | https://github.com/bigskysoftware/htmx |
| htmx extensions | https://htmx.org/extensions/ |

htmx 4 annotations (tagged `[htmx 4]`, `[htmx 4 change]`, `[htmx 4 removed]`) were sourced from the v4 beta documentation at `four.htmx.org` and the four-branch repo docs as of v4.0.0-beta5 (June 2026).
