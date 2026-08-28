---
name: htmx
description: "Implements HTMX interactions, configures swap behaviors, debugs hx-* requests, and builds hypermedia-driven UI components. Use when tasks involve hx-* attributes, HTMX AJAX requests, swap strategies, server-sent events, WebSockets, or hypermedia-driven UIs."
---

# HTMX


Use this skill for HTMX implementation and integration. Covers htmx 2.x (current stable) and htmx 4.x (beta). Reference files are annotated with `[htmx 4]`, `[htmx 4 change]`, and `[htmx 4 removed]` admonitions. Read only the reference file(s) needed for the task.

## Quick Start

1. Identify the domain of the task (attributes, requests, swapping, events, patterns).
2. Open the matching file from `references/`.
3. Implement using HTML-first, hypermedia-driven patterns.
4. Validate that server responses return HTML fragments, not JSON.

Minimal example — a button that loads content via GET:

```html
<button hx-get="/contacts" hx-target="#results" hx-swap="innerHTML">
    Load Contacts
</button>
<div id="results"></div>
```

The server endpoint (`/contacts`) must return an HTML fragment, not JSON:

```html
<!-- Server response (HTML fragment, no <html>/<body> wrappers) -->
<ul>
    <li>Alice</li>
    <li>Bob</li>
</ul>
```

The default swap is `innerHTML` — the response replaces the target's children. Use `hx-swap="outerHTML"` to replace the target element itself.

## Critical Rules

1. **HTML responses** - HTMX expects HTML responses from the server, not JSON
2. **Attribute inheritance** - (v2) Most attributes inherit to children; use `hx-disinherit` or `unset` to stop. (v4) Inheritance is explicit — use `:inherited` suffix to opt in. **Not inherited in either version:** `hx-trigger`, `hx-on*`, `hx-swap-oob`, `hx-preserve`, `hx-history-elt`, `hx-validate`
3. **Default swap is innerHTML** - Always confirm the intended swap method
4. **Form values auto-included** - Non-GET requests automatically include the closest enclosing form's values
5. **Progressive enhancement** - Use `hx-boost="true"` — pages must work without JS
6. **Escape user content** - Escape all user-supplied content server-side to prevent XSS. Wrap areas rendering user-generated content with `hx-disable` to prevent HTMX processing of injected attributes
7. **CSS lifecycle classes** - HTMX adds/removes CSS classes during requests — use for transitions and indicators
8. **data-prefix supported** - All `hx-*` attributes can also be written as `data-hx-*` for HTML validation compliance
9. **Stop polling with HTTP 286** - Server returns status `286` to stop `every` or `load delay` polling. Always use 286 for poll termination, not conditional client-side logic
10. **Error swaps** - (v2) Non-2xx responses are not swapped by default — add an `htmx:beforeSwap` listener to enable. (v4) All responses swap by default except 204/304 — use `hx-status` for per-code control
11. **Decouple with HX-Trigger headers** - Use `HX-Trigger` response headers to fire client-side events instead of hardcoding DOM element IDs in server responses
12. **Detect HTMX requests server-side** - Check the `HX-Request` header to serve HTML fragments for HTMX requests and full pages for direct browser requests. Set `Vary: HX-Request` for caching

## Reference Map

- All `hx-*` attributes, values, and modifiers: `references/attributes.md`
- Triggers, headers, parameters, CSRF, caching, CORS: `references/requests.md`
- Swap methods, targets, OOB swaps, morphing, view transitions: `references/swapping.md`
- Events, JS API, configuration, extensions, debugging: `references/events-api.md`
- Common UI patterns and examples: `references/patterns.md`
- Official extensions (WS, SSE, Idiomorph, response-targets, head-support, preload, hx-live, hx-prompt): `references/extensions.md`
- Gotchas, pitfalls, and practical guidance: `references/gotchas.md`
- Cross-file index and routing: `references/REFERENCE.md`

## Task Routing

- Adding HTMX behavior to elements -> `references/attributes.md`
- Configuring how/when requests fire -> `references/requests.md`
- Controlling where/how responses render -> `references/swapping.md`
- Handling events, JS interop, or config -> `references/events-api.md`
- Building common UI patterns (search, infinite scroll, modals, etc.) -> `references/patterns.md`
- Using WebSockets, SSE, morphing, preloading, response targeting, or head merging -> `references/extensions.md`
- Avoiding common pitfalls, accessibility, error handling, architecture decisions -> `references/gotchas.md`
- Cross-cutting concerns or architecture -> `references/REFERENCE.md` then domain-specific files
