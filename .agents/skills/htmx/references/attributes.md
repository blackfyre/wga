---
description: Complete reference for all HTMX attributes.
globs: "*.html"
---

# HTMX Attributes


All `hx-*` attributes, their accepted values, modifiers, and behavior.

> **Note:** All `hx-*` attributes can also be written as `data-hx-*` for HTML validation compliance (e.g., `data-hx-get`, `data-hx-trigger`).

## Contents

- Core Request Attributes
- Trigger Attribute
- Target & Swap Attributes
- Value & Parameter Attributes
- History Attributes
- UI Feedback Attributes
- Inheritance Control
- Security Attributes
- Miscellaneous Attributes

## Core Request Attributes

These attributes issue HTTP requests when the element is triggered.

```html
<!-- GET request -->
<button hx-get="/api/items">Load Items</button>

<!-- POST request -->
<form hx-post="/api/items">
    <input name="title" />
    <button type="submit">Create</button>
</form>

<!-- PUT request -->
<button hx-put="/api/items/1">Update</button>

<!-- PATCH request -->
<button hx-patch="/api/items/1">Patch</button>

<!-- DELETE request -->
<button hx-delete="/api/items/1">Delete</button>
```

| Attribute | HTTP Method | Notes |
|---|---|---|
| `hx-get` | GET | Parameters appended to URL as query string |
| `hx-post` | POST | Parameters sent in request body |
| `hx-put` | PUT | Parameters sent in request body |
| `hx-patch` | PATCH | Parameters sent in request body |
| `hx-delete` | DELETE | Parameters appended to URL by default |

> **[htmx 4 change]** `hx-delete` no longer includes enclosing form data by default (matches `hx-get` behavior). Use `hx-include="closest form"` if you need form values with DELETE requests.

> **[htmx 4]** New attributes `hx-action` (URL) and `hx-method` (HTTP method) provide an alternative to `hx-get`/`hx-post`/etc. Use together: `<button hx-action="/api" hx-method="post">`.

**Value:** A URL string (relative or absolute). Relative URLs resolve against the current page.

## Trigger Attribute

`hx-trigger` specifies which event(s) cause the request to fire.

> **Important:** `hx-trigger` is **NOT inherited**. It must be set on each element individually.

### Default Triggers (when `hx-trigger` is omitted)

| Element Type | Default Event |
|---|---|
| `<input>`, `<textarea>`, `<select>` | `change` |
| `<form>` | `submit` |
| Everything else | `click` |

### Syntax

```html
<!-- Single event -->
<div hx-get="/data" hx-trigger="click">Click me</div>

<!-- Multiple events (comma-separated) -->
<div hx-get="/data" hx-trigger="click, keyup">Either works</div>

<!-- Event with modifiers -->
<input hx-get="/search" hx-trigger="keyup changed delay:500ms" />

<!-- Event with filter -->
<div hx-get="/data" hx-trigger="click[ctrlKey]">Ctrl+Click only</div>

<!-- Event from another element -->
<div hx-get="/data" hx-trigger="click from:body">Body click</div>

<!-- Polling -->
<div hx-get="/status" hx-trigger="every 2s">Status: loading...</div>

<!-- Intersection observer -->
<img hx-get="/image" hx-trigger="intersect threshold:0.5" />
```

### Trigger Modifiers

| Modifier | Effect |
|---|---|
| `once` | Trigger only fires once |
| `changed` | Only fire if the element's value has changed |
| `delay:<time>` | Wait before issuing request; resets if event fires again (debounce) |
| `throttle:<time>` | Wait before issuing request; discard events during wait (throttle) |
| `from:<CSS selector>` | Listen for event on a different element. Also accepts `document`, `window`, `closest <selector>`, `find <selector>`, `next`, `next <selector>`, `previous`, `previous <selector>`. **Note:** Selectors with whitespace require parentheses: `from:(form input)` |
| `target:<CSS selector>` | Only fire if the event target matches the selector |
| `consume` | Prevent event from propagating to parent elements |
| `queue:<strategy>` | Queue behavior when event fires during an active request. Values: `first`, `last` (default), `all`, `none` |

> **[htmx 4 removed]** The `queue` trigger modifier is removed. Use `hx-sync` instead: `<div hx-trigger="click" hx-get="/test" hx-sync="this:queue all">`.

### Trigger Filters

JavaScript expressions inside square brackets. The event object is available as `event`. Standard event properties work: `ctrlKey`, `shiftKey`, `altKey`, `metaKey`, `key`, etc.

```html
<!-- Only on Ctrl+Click -->
<div hx-get="/data" hx-trigger="click[ctrlKey]">Ctrl+Click</div>

<!-- Only on Enter key -->
<input hx-get="/search" hx-trigger="keyup[key=='Enter']" />

<!-- Custom condition -->
<div hx-get="/data" hx-trigger="click[checkCondition()]">Conditional</div>
```

### Special Triggers

| Trigger | When It Fires |
|---|---|
| `load` | When the element is loaded into the DOM |
| `revealed` | When the element first scrolls into the viewport |
| `intersect` | When the element intersects the viewport. Options: `root:<selector>`, `threshold:<float>` |
| `every <time>` | Polls at the given interval (e.g., `every 2s`, `every 500ms`) |

**Stopping polling:** The server responds with HTTP status `286` to signal the client to stop polling.

### Load Polling

Combine `load` trigger with `delay` for server-driven polling:

```html
<div hx-get="/status" hx-trigger="load delay:1s" hx-swap="outerHTML">
    Checking status...
</div>
```

The server returns a new element with the same trigger to continue, or without it to stop.

## Target & Swap Attributes

### hx-target

Specifies which element receives the response content.

```html
<!-- CSS selector -->
<button hx-get="/data" hx-target="#results">Load</button>

<!-- this (current element) -->
<button hx-get="/data" hx-target="this">Replace me</button>

<!-- Extended selectors -->
<button hx-get="/data" hx-target="closest div">Nearest div ancestor</button>
<button hx-get="/data" hx-target="find .content">First .content descendant</button>
<button hx-get="/data" hx-target="next .sibling">Next .sibling in DOM</button>
<button hx-get="/data" hx-target="previous .sibling">Previous .sibling in DOM</button>
```

| Extended Selector | Meaning |
|---|---|
| `this` | The element itself |
| `closest <selector>` | Nearest ancestor matching selector |
| `find <selector>` | First descendant matching selector |
| `next` | Next sibling element |
| `next <selector>` | Next element in DOM matching selector |
| `previous` | Previous sibling element |
| `previous <selector>` | Previous element in DOM matching selector |

**Default:** Without `hx-target`, the element that made the request is the target.

### hx-swap

Controls how the response content is inserted into the target. See `references/swapping.md` for full details.

**Values:** `innerHTML` (default), `outerHTML`, `textContent`, `beforebegin`, `afterbegin`, `beforeend`, `afterend`, `delete`, `none`

### hx-select

Pick a subset of the response to swap in using a CSS selector.

```html
<!-- Only swap in the #results element from the response -->
<button hx-get="/page" hx-select="#results" hx-target="#results">Load</button>
```

### hx-select-oob

Pick specific elements from the response for out-of-band swaps.

```html
<!-- Swap #results into target, and also OOB swap #notification -->
<button hx-get="/page" hx-select="#results" hx-select-oob="#notification">Load</button>
```

## Value & Parameter Attributes

### hx-vals

Add extra values to the request as a JSON object.

```html
<!-- Static JSON values -->
<button hx-post="/action" hx-vals='{"key": "value", "count": 42}'>Submit</button>

<!-- Dynamic values with js: prefix -->
<button hx-post="/action" hx-vals='js:{timestamp: Date.now()}'>Submit</button>
```

> **[htmx 4]** `js:` expressions in `hx-vals`, `hx-headers`, and `hx-confirm` receive the request context object `ctx` in scope (beta5) — e.g. `hx-vals='js:{method: ctx.request.method}'`. Expressions may return a Promise for async value resolution.

### hx-vars (deprecated — use hx-vals with js: prefix)

> **[htmx 4 removed]** `hx-vars` is removed. Use `hx-vals` with `js:` prefix instead.

Comma-separated name-expression pairs evaluated as JavaScript.

### hx-include

Include values from additional elements in the request.

```html
<!-- Include specific input -->
<button hx-post="/action" hx-include="#extra-field">Submit</button>

<!-- Include all inputs in a container -->
<button hx-post="/action" hx-include="#sidebar">Submit</button>

<!-- Extended selectors work -->
<button hx-post="/action" hx-include="closest form">Submit</button>
```

### hx-params

> **[htmx 4 removed]** `hx-params` is removed. Use the `htmx:config:request` event to filter parameters programmatically.

Filter which parameters are submitted.

```html
<!-- Only include listed params -->
<form hx-post="/action" hx-params="name, email">...</form>

<!-- Exclude listed params -->
<form hx-post="/action" hx-params="not password">...</form>

<!-- Include all (default) -->
<form hx-post="/action" hx-params="*">...</form>

<!-- Include none -->
<form hx-post="/action" hx-params="none">...</form>
```

### hx-encoding

Set the encoding type for the request.

> **[htmx 4]** When `hx-encoding` is absent, htmx falls back to the form's native `enctype` attribute (beta5) — `<form enctype="multipart/form-data">` works without duplication.

```html
<!-- Required for file uploads -->
<form hx-post="/upload" hx-encoding="multipart/form-data">
    <input type="file" name="document" />
    <button type="submit">Upload</button>
</form>
```

### hx-headers

Add custom headers to the request as a JSON object.

```html
<!-- Static headers -->
<div hx-get="/api" hx-headers='{"X-Custom": "value"}'>Load</div>

<!-- Dynamic headers with js: prefix -->
<div hx-get="/api" hx-headers='js:{"Authorization": "Bearer " + getToken()}'>Load</div>
```

## History Attributes

### hx-push-url

Push a URL into the browser's history stack after a successful request.

```html
<!-- Push the request URL -->
<a hx-get="/page" hx-push-url="true">Navigate</a>

<!-- Push a custom URL -->
<a hx-get="/api/page" hx-push-url="/page">Navigate</a>

<!-- Disable (override inherited) -->
<a hx-get="/page" hx-push-url="false">No history</a>
```

### hx-replace-url

Replace the current URL in the browser's location bar (no new history entry).

```html
<a hx-get="/page" hx-replace-url="true">Update URL</a>
<a hx-get="/api/page" hx-replace-url="/page">Custom URL</a>
```

### hx-history

> **[htmx 4 removed]** `hx-history` is removed. htmx 4 no longer caches pages in localStorage — back/forward navigation re-fetches from the server. Configure via `htmx.config.history = false` (disable) or `htmx.config.history = "reload"` (full page reload).

Prevent the current page from being saved in the history cache.

```html
<!-- On the page's body or a top-level element -->
<body hx-history="false">
    <!-- Sensitive content not cached -->
</body>
```

### hx-history-elt

Specify which element should be snapshot and restored during history navigation (default: `<body>`). This element **must always be visible** in the application for proper history restoration. Not inherited.

```html
<div id="content" hx-history-elt>
    <!-- Only this element is cached/restored -->
</div>
```

## UI Feedback Attributes

### hx-indicator

Specify the element that gets the `htmx-request` class during requests, making indicators visible.

```html
<button hx-get="/data" hx-indicator="#spinner">
    Load
</button>
<span id="spinner" class="htmx-indicator">Loading...</span>
```

The `htmx-indicator` class sets `opacity: 0` by default. When the parent (or hx-indicator target) receives `htmx-request`, the indicator becomes visible (`opacity: 1`).

Multiple indicators:

```html
<button hx-get="/data" hx-indicator=".my-indicator">Load</button>
```

### hx-disabled-elt

> **[htmx 4 change]** Renamed to `hx-disable`. Rename in order: first rename security `hx-disable` → `hx-ignore`, then rename `hx-disabled-elt` → `hx-disable`.

Add the `disabled` attribute to elements while a request is in flight.

```html
<!-- Disable this element -->
<button hx-post="/action" hx-disabled-elt="this">Submit</button>

<!-- Disable another element -->
<button hx-post="/action" hx-disabled-elt="#other-btn">Submit</button>

<!-- Disable multiple elements -->
<button hx-post="/action" hx-disabled-elt="closest form, #other-btn">Submit</button>
```

### hx-confirm

Show a browser `confirm()` dialog before issuing the request.

```html
<button hx-delete="/item/1" hx-confirm="Are you sure?">Delete</button>
```

Use `hx-confirm="unset"` on a child to disable an inherited confirm.

> **[htmx 4]** `hx-confirm` accepts a `js:` expression (beta5) evaluated with `ctx` in scope — a truthy result proceeds, replacing `window.confirm`. The `htmx:confirm` event detail carries `ctx` plus `issueRequest()` / `dropRequest()` callbacks for async custom dialogs.

### hx-prompt

> **[htmx 4 change]** `hx-prompt` is removed from core but restored by the official `hx-prompt` extension (beta5). Load `dist/ext/hx-prompt.js` and the attribute works as in v2, including the `HX-Prompt` request header. Cancel aborts the request; a cancelable `htmx:prompt` event fires after a valid answer; assign `window.htmxPrompt = (question) => answer` for a custom synchronous dialog. Without the extension, use `hx-on::config:request="ctx.request.headers['HX-Prompt'] = prompt('...') ?? event.preventDefault()"`.

Show a browser `prompt()` dialog. The user's response is sent in the `HX-Prompt` request header.

```html
<button hx-post="/rename" hx-prompt="Enter new name:">Rename</button>
```

## Inheritance Control

Most `hx-*` attributes inherit from parent to child elements. **Not inherited:** `hx-trigger`, `hx-on*`, `hx-swap-oob`, `hx-preserve`, `hx-history-elt`, `hx-validate`.

> **[htmx 4 change]** Inheritance is **explicit** by default. Attributes no longer cascade to children unless you add the `:inherited` suffix: `hx-confirm:inherited="Are you sure?"`. Use `:inherited:append` to add to (rather than replace) inherited values. Restore v2 implicit inheritance with `htmx.config.implicitInheritance = true`.

### hx-disinherit

> **[htmx 4 removed]** `hx-disinherit` is removed. Inheritance is explicit in v4 — no need to opt out since attributes don't cascade by default.

Disable inheritance of specific attributes for child elements.

```html
<!-- Disable all inheritance -->
<div hx-confirm="Sure?" hx-disinherit="*">
    <button hx-delete="/item/1">No confirm dialog</button>
</div>

<!-- Disable specific attribute inheritance -->
<div hx-target="#results" hx-disinherit="hx-target">
    <button hx-get="/data">Uses own default target</button>
</div>
```

### hx-inherit

> **[htmx 4 removed]** `hx-inherit` is removed. Use the `:inherited` suffix on individual attributes instead: `hx-target:inherited="#results"`.

Explicitly enable inheritance for specific attributes (useful when `htmx.config.disableInheritance = true`).

```html
<div hx-target="#results" hx-inherit="hx-target">
    <button hx-get="/data">Inherits hx-target</button>
</div>
```

### Unsetting Inherited Values

Use `"unset"` as the value to clear an inherited attribute:

```html
<div hx-confirm="Sure?">
    <button hx-get="/safe" hx-confirm="unset">No confirm</button>
</div>
```

## Security Attributes

### hx-disable

> **[htmx 4 change]** Renamed to `hx-ignore`. The name `hx-disable` is repurposed for the old `hx-disabled-elt` behavior (disabling form elements during requests).

Completely prevents HTMX processing on the element and all its children. Cannot be overridden by injected content.

```html
<div hx-disable>
    <!-- No HTMX processing here, even if attributes are present -->
    <button hx-get="/data">This will NOT make a request</button>
</div>
```

## Miscellaneous Attributes

### hx-boost

Convert standard anchors and forms into AJAX requests targeting the `<body>`.

```html
<!-- Boost all links and forms in this container -->
<div hx-boost="true">
    <a href="/page">AJAX navigation</a>
    <form action="/submit" method="post">
        <button type="submit">AJAX submit</button>
    </form>
</div>
```

Boosted anchors push the URL into browser history. **Boosted forms do NOT push URL** — add `hx-push-url="true"` explicitly if needed. Boosted pages must serve full HTML pages. They degrade gracefully when JS is disabled. Only same-domain links are boosted; local anchor links are ignored.

> **[htmx 4]** Links with a `download` attribute are never boosted (beta5) — the browser's native download behavior is preserved.

### hx-ext

> **[htmx 4 removed]** `hx-ext` is removed. Extensions auto-register when their script is loaded. Restrict allowed extensions with `<meta name="htmx-config" content='{"extensions": "sse, ws"}'>`.

Enable HTMX extensions on an element and its children.

```html
<!-- Single extension -->
<div hx-ext="preload">...</div>

<!-- Multiple extensions -->
<div hx-ext="preload, response-targets">...</div>

<!-- Disable an inherited extension -->
<div hx-ext="ignore:preload">...</div>
```

### hx-preserve

Keep an element unchanged across swaps. The element must have an `id`. Useful for video players, iframes, or complex widgets.

```html
<video id="player" hx-preserve>
    <source src="video.mp4" />
</video>
```

### hx-sync

Control how requests from multiple elements are synchronized.

```html
<!-- Abort this request if the form submits -->
<input hx-get="/search" hx-trigger="keyup" hx-sync="closest form:abort" />

<!-- Drop new requests while one is in flight -->
<button hx-post="/action" hx-sync="this:drop">Submit</button>

<!-- Queue requests, keep last -->
<input hx-get="/search" hx-sync="this:queue last" />
```

| Strategy | Behavior |
|---|---|
| `drop` | Drop new request if one is in flight |
| `abort` | Abort the current request and issue new one |
| `replace` | Abort current request, issue new one (alias for `abort`) |
| `queue first` | Queue first request, drop subsequent |
| `queue last` | Queue most recent request, drop earlier queued |
| `queue all` | Queue all requests |

### hx-validate

Force elements to validate themselves before a request is issued.

```html
<input hx-get="/check" hx-validate="true" required pattern="[a-z]+" />
```

### hx-request

> **[htmx 4 removed]** `hx-request` is removed. Use `hx-config` for per-element request configuration instead.

Configure various aspects of the request.

```html
<!-- Set timeout -->
<button hx-get="/slow" hx-request='"timeout": 5000'>Load</button>

<!-- Disable credentials -->
<button hx-get="/api" hx-request='"credentials": false'>Load</button>

<!-- Disable no-cache header -->
<button hx-get="/api" hx-request='"noHeaders": true'>Load</button>
```

### hx-on

Respond to any event directly with inline JavaScript. Uses `hx-on:<event-name>` or `hx-on-<event-name>` syntax (colon or dash separator).

```html
<!-- Standard DOM events -->
<button hx-on:click="alert('clicked!')">Click</button>

<!-- HTMX events (use kebab-case for camelCase events) -->
<form hx-post="/submit" hx-on:htmx:before-request="showSpinner()">
    <button type="submit">Submit</button>
</form>

<!-- Modify request parameters -->
<div hx-get="/data" hx-on:htmx:config-request="event.detail.parameters.extra = 'value'">
    Load
</div>
```

**Note:** HTML attributes are case-insensitive. Use kebab-case for camelCase event names (e.g., `htmx:config-request` instead of `htmx:configRequest`).

## htmx 4 Attributes

> **[htmx 4]** The following attributes are only available in htmx 4.

### hx-status

Control swap behavior per HTTP status code. Accepts exact codes (`404`), single-digit wildcards (`50x`), and range wildcards (`5xx`). More specific codes take priority.

```html
<form hx-post="/save"
    hx-status:422="swap:innerHTML target:#errors select:#validation-errors"
    hx-status:5xx="swap:none push:false">
</form>
```

Available config keys per status rule: `swap:`, `target:`, `select:`, `push:`, `replace:`, `transition:`.

### hx-config

Per-element request configuration in JSON or `key:value` syntax. Replaces `hx-request`.

### hx-validate

Control form validation behavior on the element.

### hx-ignore

Replaces v2's `hx-disable`. Completely prevents htmx processing on the element and all its children.

```html
<div hx-ignore>
    <!-- No htmx processing here -->
    {{ user_generated_content }}
</div>
```

## Sources

[^1]: htmx attribute reference. <https://htmx.org/reference/#attributes> — complete list of core and additional attributes with descriptions.

[^2]: htmx documentation. <https://htmx.org/docs/> — attribute inheritance, triggering, swapping, and request behavior.

[^3]: htmx 4 migration guide. <https://four.htmx.org/docs/get-started/migration/> — attribute removals (`hx-params`, `hx-disinherit`, `hx-ext`, `hx-request`, `hx-history`; `hx-prompt` restored via extension in beta5), new attributes (`hx-action`, `hx-method`, `hx-status`, `hx-config`, `hx-ignore`), and explicit inheritance via `:inherited` suffix.

[^4]: htmx GitHub repository. <https://github.com/bigskysoftware/htmx> — source code and per-attribute behavior.
