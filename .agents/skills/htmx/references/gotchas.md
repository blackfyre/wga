---
description: Practical gotchas, common pitfalls, and field-tested guidance for getting the most out of HTMX.
globs: "*.html"
---

# HTMX Gotchas & Practical Guidance


Hard-won lessons from production HTMX projects. Use this reference to avoid common pitfalls and make informed architectural decisions.

## Contents

- Where HTMX Excels
- Silent Failures
- Error Handling
- Accessibility
- Attribute Inheritance Surprises
- hx-boost Pitfalls
- Client-Side Interactions
- Endpoint Design
- Testing
- Performance Considerations
- Mixing with SPA Frameworks
- Migration & Lock-In
- Version Awareness

## Where HTMX Excels

HTMX is at its best for:

- **CRUD apps, dashboards, and content-driven sites** — server-rendered HTML with targeted dynamic updates
- **Progressive enhancement** of traditional multi-page apps — add interactivity without rewriting
- **Teams that are backend-focused** — most logic stays in Python/Go/Ruby/PHP, minimal JS context-switching
- **Projects that value long-term stability** — no build step, no package manager, stable API

The sweet spot is **enhancing parts of a page** (form submissions, table updates, search) rather than building a full SPA.

## Silent Failures

HTMX fails silently in several common scenarios. These are the most frequent sources of "nothing happened" bugs.

### Missing or mistyped target selectors

If `hx-target` references an ID that doesn't exist, HTMX fires an `htmx:targetError` event but produces no visible feedback — the response is silently dropped.

```html
<!-- BUG: #resutls is a typo — HTMX silently drops the response -->
<button hx-get="/search" hx-target="#resutls">Search</button>
<div id="results"></div>
```

**Fix:** Use `htmx.logAll()` during development to see all events. Add an event listener to surface target errors:

```javascript
document.addEventListener('htmx:targetError', function(event) {
    console.error('HTMX target not found:', event.detail.target);
});
```

### Server errors (5xx) produce no visible feedback (htmx 2.x)

In htmx 2.x, when a server returns a 4xx/5xx error, the response is **not swapped** — the user sees no indication that their action failed. (Note: htmx 4.0 reverses this — all responses swap by default except 204 and 304.)

```javascript
// Add global error handling
document.addEventListener('htmx:responseError', function(event) {
    var status = event.detail.xhr.status;
    var elt = event.detail.elt;
    if (status >= 500) {
        elt.innerHTML = '<div class="error">Something went wrong. Please try again.</div>';
    }
});
```

Or use the `response-targets` extension to route errors to specific elements:

```html
<div hx-ext="response-targets">
    <button hx-get="/data"
            hx-target="#content"
            hx-target-500="#error-message">
        Load
    </button>
    <div id="content"></div>
    <div id="error-message"></div>
</div>
```

### Removed type="submit" breaks forms

HTML minifiers or template engines sometimes strip `type="submit"` from buttons. HTMX may not detect the form submission correctly without it.

**Fix:** Always explicitly set `type="submit"` on form submit buttons, and test after enabling minification.

## Error Handling

### Add loading states and disable controls during requests

HTMX's default request queuing can surprise users — by default, if a request is in-flight, new triggering events are **dropped** (ignored). Only the last queued event is retained. Use `hx-sync` to customize this behavior.

```html
<button hx-post="/submit"
        hx-indicator="#spinner"
        hx-disabled-elt="this">
    Submit
    <span id="spinner" class="htmx-indicator">...</span>
</button>
```

### Use htmx:beforeSwap for conditional error handling

```javascript
document.addEventListener('htmx:beforeSwap', function(event) {
    // Allow 422 responses to swap (for validation errors)
    if (event.detail.xhr.status === 422) {
        event.detail.shouldSwap = true;
        event.detail.isError = false;
    }
});
```

## Accessibility

HTMX does not automatically manage accessibility. Dynamic content swaps can break screen reader announcements, focus management, and keyboard navigation.

### Data: htmx sites score lower on accessibility audits

A [Wagtail CMS analysis of HTTP Archive data](https://wagtail.org/blog/htmx-accessibility-gaps-data-and-recommendations/) found that since November 2024, htmx-powered sites score below the cross-technology average on Lighthouse accessibility checks. The most common failures on htmx sites:

- **`link-name`** — links lacking descriptive text (significantly more prevalent on htmx sites)
- **`heading-order`** — headings not in logical sequence
- **`aria-allowed-role`** — HTML elements assigned incompatible ARIA roles

Even official htmx examples contribute: the Progress Bar example uses `<h3 role="status">` — an `h3` cannot hold the `status` role, and the heading level may be out of sequence.

This isn't inherent to htmx — it's a gap in documentation and examples. The fix is to follow [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/) when implementing interactive patterns.

### Use semantic elements — not divs with hx-get

```html
<!-- BAD: not focusable, not announced by screen readers -->
<div hx-get="/next-page">Next</div>

<!-- BAD: anchor without href is not focusable -->
<a hx-get="/next-page">Next</a>

<!-- GOOD: proper button element -->
<button hx-get="/next-page" hx-target="#content">Next</button>

<!-- GOOD: anchor with href for progressive enhancement -->
<a href="/next-page" hx-get="/next-page" hx-target="#content">Next</a>
```

### Don't use hx-post/hx-delete on anchor tags

Anchors (`<a>`) are semantically for navigation (GET). Using `hx-post` or `hx-delete` on them confuses assistive technologies. Use `<button>` for actions.

### Manage focus after swaps

When content is swapped, the focused element may be removed from the DOM, leaving focus in limbo.

```javascript
document.addEventListener('htmx:afterSwap', function(event) {
    // Focus the first focusable element in the swapped content
    var focusable = event.detail.target.querySelector(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    if (focusable) focusable.focus();
});
```

### Announce dynamic updates to screen readers

```html
<!-- Add aria-live to regions that update dynamically -->
<div id="search-results" aria-live="polite" aria-atomic="false">
    <!-- HTMX swaps content here -->
</div>
```

## Attribute Inheritance Surprises

Most `hx-*` attributes inherit from parent to child elements. This is powerful but can cause unexpected behavior.

### Problem: a child unintentionally inherits a parent's target

```html
<!-- Parent sets target for its own button -->
<div hx-target="#main-content">
    <button hx-get="/page">Load Page</button>

    <!-- BUG: this nested component inherits hx-target="#main-content" -->
    <div class="widget">
        <button hx-get="/widget-data">Refresh Widget</button>
    </div>
</div>
```

**Fix:** Use `hx-target` explicitly on the child, or use `hx-disinherit` to block inheritance:

```html
<div class="widget" hx-disinherit="hx-target">
    <button hx-get="/widget-data" hx-target="#widget-content">Refresh Widget</button>
</div>
```

### Attributes that do NOT inherit

These are safe from inheritance surprises: `hx-trigger`, `hx-on*`, `hx-swap-oob`, `hx-preserve`, `hx-history-elt`, `hx-validate`.

## hx-boost Pitfalls

`hx-boost` converts standard links and forms into AJAX requests. It's great for progressive enhancement but causes problems when overused.

### Don't boost entire pages or the body element

```html
<!-- RISKY: boosting everything causes hard-to-debug issues -->
<body hx-boost="true">
    <!-- Every link and form is now AJAX — including ones that shouldn't be -->
</body>
```

**Problems with global boost:**
- Refresh/reload can show blank pages if history caching misbehaves
- Links to external sites, file downloads, or different apps break
- Forms with file uploads may not work as expected

**Fix:** Boost specific containers:

```html
<nav hx-boost="true">
    <a href="/dashboard">Dashboard</a>
    <a href="/settings">Settings</a>
</nav>

<!-- External links, downloads, etc. stay outside boosted containers -->
<a href="https://external-site.com">External Link</a>
<a href="/files/report.pdf" download>Download Report</a>
```

### Always set hx-target when boosting

Without an explicit target, boosted requests replace the entire `<body>` innerHTML (the default). For partial page updates, set an explicit target:

```html
<nav hx-boost="true" hx-target="#main-content" hx-select="#main-content">
    <a href="/dashboard">Dashboard</a>
</nav>
```

### Serve fragments for HX-Request, full pages otherwise

```python
# Server-side: detect HTMX requests
if request.headers.get('HX-Request'):
    return render_template('dashboard_fragment.html')
else:
    return render_template('dashboard_full.html')
```

## Client-Side Interactions

Not every interaction needs a server round-trip. Toggling visibility, showing/hiding elements, and updating counters should happen client-side.

### Use Alpine.js or vanilla JS for pure UI state

```html
<!-- BAD: server round-trip just to toggle a menu -->
<button hx-get="/toggle-menu" hx-target="#menu">Toggle</button>

<!-- GOOD: client-side toggle with Alpine.js -->
<div x-data="{ open: false }">
    <button @click="open = !open">Toggle</button>
    <nav x-show="open">Menu items...</nav>
</div>

<!-- GOOD: client-side toggle with vanilla JS -->
<button onclick="document.getElementById('menu').toggleAttribute('hidden')">
    Toggle
</button>
<nav id="menu" hidden>Menu items...</nav>
```

### Guideline: if the interaction doesn't change server state, keep it client-side

Common examples: accordions, tabs (when content is already loaded), tooltips, dropdowns, form field show/hide.

## Endpoint Design

HTMX projects tend to need more server endpoints than traditional MPAs because each dynamic region may need its own fragment endpoint.

### Design endpoints that serve both full pages and fragments

Use the `HX-Request` header to detect HTMX requests:

```python
@app.route('/contacts')
def contacts():
    contacts = get_contacts()
    if request.headers.get('HX-Request'):
        return render_template('contacts_table.html', contacts=contacts)
    return render_template('contacts_page.html', contacts=contacts)
```

### Set the Vary header for proper caching

If the same URL returns different content based on `HX-Request`, caches must know:

```
Vary: HX-Request
```

Without this, a CDN or browser cache may serve an HTML fragment as a full page (or vice versa).

### Avoid coupling endpoints to specific UI locations

```python
# BAD: endpoint knows which sidebar element to update
@app.route('/add-item', methods=['POST'])
def add_item():
    item = create_item(request.form)
    # This endpoint "knows" about #sidebar-count — fragile
    return render_template('item.html', item=item), 200, {
        'HX-Trigger': 'itemAdded'
    }
```

Prefer using `HX-Trigger` response headers to decouple — let the client-side markup decide what to refresh when an event fires.

## Testing

HTMX behavior is difficult to unit test because logic is split between server-rendered markup and client-side attribute-driven behavior.

### Recommended testing strategy

1. **Server tests:** Verify endpoints return correct HTML fragments with proper structure and content
2. **End-to-end tests:** Use Playwright or Cypress to test the full interaction — click, request, swap, result
3. **Attribute audits:** Periodically grep for `hx-target` values and verify the referenced IDs exist in the corresponding pages

```bash
# Quick audit: find all hx-target values and check they have matching IDs
grep -roh 'hx-target="[^"]*"' templates/ | sort -u
grep -roh 'id="[^"]*"' templates/ | sort -u
```

### Watch for minifier/template engine interference

HTML minifiers may strip attributes that HTMX depends on (like `type="submit"`). Always test after enabling minification.

## Performance Considerations

### Every state change requires a network round-trip

For rapid interactions (typing, dragging, real-time updates), the network latency can make the UI feel sluggish. HTMX is not the right tool for:

- Real-time collaborative editing
- Drag-and-drop with instant visual feedback (use client-side JS, sync on drop)
- Keystroke-by-keystroke validation (debounce aggressively with `delay:500ms` or validate client-side)

### First load advantage

HTMX pages typically have faster First Contentful Paint and Largest Contentful Paint than client-side rendered SPAs because meaningful HTML arrives in the initial response. This is a genuine architectural advantage for content-heavy sites.

### Debounce and deduplicate

```html
<!-- Use delay and changed modifier to avoid excessive requests -->
<input hx-get="/search"
       hx-trigger="input changed delay:300ms"
       hx-target="#results"
       name="q" />
```

## Mixing with SPA Frameworks

Avoid combining HTMX with React, Vue, or other frameworks that manage their own DOM.

**Why it breaks:**
- HTMX swaps DOM nodes directly — React/Vue won't know about the changes and will overwrite them on next render
- Event systems may conflict
- State management becomes split between server (HTMX) and client (framework), creating the exact duplication HTMX was meant to eliminate

**If you must integrate:** Keep HTMX and the SPA framework in completely separate DOM regions that never overlap.

## Migration & Lock-In

`hx-*` attributes are HTMX-specific. Adopting HTMX means your markup depends on the library.

### Perspective

- Migrating away from HTMX requires rewriting markup, but the server-side logic (endpoints returning HTML) is standard and reusable
- This is comparable to migrating away from any framework — React JSX, Vue templates, and Angular directives all create similar coupling
- HTMX projects tend to have less total code than SPA equivalents, so the migration surface is often smaller

### Reduce coupling where possible

- Use `data-hx-*` attribute format for HTML validation compliance
- Keep endpoint responses as standard HTML fragments that could be consumed by other clients
- Use progressive enhancement (`hx-boost` with `href` fallbacks) so pages work without HTMX

## Version Awareness

### HTMX 2.x to 4.0 changes (The Fetchening)

htmx 4.0 is in beta (v4.0.0-beta5, released 2026-06-26; beta3 was designated release candidate 1). Current stable: htmx 2.0.10.

**Quick restore of v2 behavior:**

```javascript
htmx.config.implicitInheritance = true;
htmx.config.noSwap = [204, 304, '4xx', '5xx'];
```

Or load the `htmx-2-compat` extension which restores implicit inheritance, legacy event names, and error-response defaults.

**Upgrade checker:** `npx htmx.org@next upgrade-check -- ./path/to/project` scans templates and JS for v2 code that needs updating.

### Breaking changes

- **`fetch()` replaces `XMLHttpRequest`** — irreversible. File upload progress events (`htmx:xhr:progress`) are removed. Streaming via `ReadableStream` is now possible
- **Attribute inheritance is explicit** — use `:inherited` suffix (`hx-target:inherited="#output"`) or set `htmx.config.implicitInheritance = true`
- **Error responses swap by default** — 4xx/5xx responses are swapped into the DOM (only 204 and 304 excluded). Revert with `htmx.config.noSwap = [204, 304, '4xx', '5xx']`. Use `hx-status` for per-code control
- **Event names use colon-separated format** — `htmx:afterSwap` → `htmx:after:swap`, `htmx:configRequest` → `htmx:config:request`. Multiple error events consolidated into `htmx:error`
- **Local history caching removed** — back/forward navigation re-fetches from server instead of localStorage snapshots
- **`hx-delete` excludes form data** — matches `hx-get` behavior. Use `hx-include="closest form"` if needed
- **OOB swap order changed** — main content swaps first, then OOB elements (was reversed in v2)
- **Default timeout is 60 seconds** — was 0 (no timeout). Revert with `htmx.config.defaultTimeout = 0`
- **`queue` trigger modifier removed** — use `hx-sync` instead

### Renamed attributes

| v2 | v4 |
|---|---|
| `hx-disable` (security) | `hx-ignore` |
| `hx-disabled-elt` | `hx-disable` |

### Removed attributes

`hx-vars` (use `hx-vals` with `js:`), `hx-params` (use `htmx:config:request` event), `hx-prompt` (restored by the official `hx-prompt` extension as of beta5; without it, use `hx-confirm` with `js:`), `hx-ext` (extensions auto-register), `hx-disinherit` / `hx-inherit` (inheritance is explicit), `hx-request` (use `hx-config`), `hx-history` (no localStorage caching)

### New features

- **`hx-status`** — per-element status code swap control: `hx-status:422="swap:innerHTML target:#errors"`
- **`<hx-partial>`** — cleaner multi-target responses replacing `hx-swap-oob`
- **`innerMorph` / `outerMorph` swap strategies** — Idiomorph in core, no extension needed
- **`textContent` and `delete` swap strategies** — in core
- **`hx-action` + `hx-method`** — alternative to verb-specific attributes
- **`hx-config`** — per-element request configuration
- **Short swap aliases** — `before`, `after`, `prepend`, `append`
- **JSX compatibility** — `htmx.config.metaCharacter = "-"` replaces `:` with `-` in attribute names
- **`swapEmpty` swap modifier + `htmx.config.defaultSwapEmpty`** (beta5) — control whether an empty response body clears the target; SSE messages default to not swapping empty responses
- **Declarative morph freezing** (beta5) — `hx-morph-skip` / `hx-morph-skip-children` attributes in server templates freeze elements (or their children) during `innerMorph`/`outerMorph`
- **`ctx` in `js:` expressions** (beta5) — `hx-vals`, `hx-headers`, and `hx-confirm` `js:` expressions receive the request context object
- **HCON** — htmx Configuration Object Notation, the formalized mini-language for structured attributes (`key:value` pairs, bare boolean flags, dotted nesting, JSON fallback for values starting with `{`)

HTMX 2.x will continue to be supported. No rush to migrate, but be aware when starting new projects. Each reference file in this skill is annotated with `[htmx 4]`, `[htmx 4 change]`, and `[htmx 4 removed]` admonitions at the relevant sections.

## Sources

[^1]: htmx documentation. <https://htmx.org/docs/> — security, accessibility, and integration guidance.

[^2]: htmx documentation — security considerations. <https://htmx.org/docs/#security>

[^3]: htmx 4 migration guide — backward compatibility layer (`htmx-2-compat` extension), behavioral changes, removed attributes and config options. <https://four.htmx.org/docs/get-started/migration/>

[^4]: htmx GitHub repository. <https://github.com/bigskysoftware/htmx> — issue tracker and source for edge-case behavior.
