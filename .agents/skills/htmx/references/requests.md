---
description: HTMX request lifecycle, headers, parameters, security, and caching.
globs: "*.html"
---

# HTMX Requests


How HTMX issues requests, what gets sent, and how to control the lifecycle.

## Contents

- Request Lifecycle
- Request Headers
- Response Headers
- Parameters & Values
- File Upload
- CSRF Protection
- CORS
- Caching
- Security
- Validation
- Confirmation & Prompts

## Request Lifecycle

The order of operations for an HTMX request:

1. Element is triggered (event fires, passes filters)
2. Values are gathered from the element and included elements
3. `htmx-request` class is applied to the element (or `hx-indicator` target)
4. AJAX request is issued
5. Upon response, `htmx-swapping` class is added to the target
6. Optional swap delay (`swap:<time>` modifier)
7. Old content is swapped out, new content is swapped in
8. `htmx-settling` class is applied to the target
9. Settle delay (default 20ms)
10. DOM is settled, `htmx-added` class removed from new content
11. All HTMX classes are removed

### CSS Classes During Lifecycle

| Class | Applied To | When |
|---|---|---|
| `htmx-request` | Triggering element or `hx-indicator` target | During the request |
| `htmx-swapping` | Target element | After response, before swap |
| `htmx-settling` | Target element | After swap, before settle completes |
| `htmx-added` | New content elements | After swap, removed after settle |

Use these classes for loading indicators, animations, and transitions.

## Request Headers

HTMX automatically includes these headers with every request:

> **[htmx 4 change]** `HX-Trigger` → `HX-Source` (format changed to `tagName#id`). `HX-Target` format changed to `tagName#id`. `HX-Trigger-Name` is removed; `HX-Prompt` is removed from core but restored by the `hx-prompt` extension (beta5). New headers: `HX-Request-Type` (`"full"` or `"partial"`), `Accept: text/html`.

| Header | Value | Purpose |
|---|---|---|
| `HX-Boosted` | `true` | Present when request is via `hx-boost` |
| `HX-Current-URL` | Current browser URL | Lets server know the user's current page |
| `HX-History-Restore-Request` | `true` | Present when request is for history cache miss restoration |
| `HX-Prompt` | User's response | Present when `hx-prompt` was used. **[htmx 4 change]** Removed from core; sent by the `hx-prompt` extension (beta5) |
| `HX-Request` | `true` | Always present on HTMX requests |
| `HX-Target` | Target element's `id` | The target element for the response |
| `HX-Trigger` | Triggering element's `id` | The element that triggered the request. **[htmx 4 change]** Renamed to `HX-Source` |
| `HX-Trigger-Name` | Triggering element's `name` | The `name` attribute of the triggering element. **[htmx 4 removed]** |

### Server-Side Detection

Use `HX-Request: true` to detect HTMX requests and return partial HTML instead of full pages:

```python
# Python/Flask example
if request.headers.get('HX-Request'):
    return render_template('partial.html')
return render_template('full_page.html')
```

```javascript
// Node.js/Express example
if (req.headers['hx-request']) {
    return res.render('partial');
}
res.render('full_page');
```

## Response Headers

The server can control client behavior using these response headers:

| Header | Effect |
|---|---|
| `HX-Location` | Client-side redirect (AJAX, no full page reload). Value: URL string or JSON `{"path": "/url", "target": "#el"}` |
| `HX-Push-Url` | Push URL into browser history. Value: URL string or `false` |
| `HX-Redirect` | Full-page client-side redirect (non-AJAX) |
| `HX-Refresh` | Full page refresh. Value: `true` |
| `HX-Replace-Url` | Replace current URL in location bar (no new history entry). Value: URL string or `false` |
| `HX-Reswap` | Override the `hx-swap` value for this response. Value: any valid swap strategy |
| `HX-Retarget` | Override the `hx-target` value. Value: CSS selector |
| `HX-Reselect` | Override the `hx-select` value. Value: CSS selector |
| `HX-Trigger` | Trigger client-side events. Value: event name or JSON `{"event": {"key": "value"}}` |
| `HX-Trigger-After-Settle` | Trigger events after the settle phase. **[htmx 4 removed]** — use `HX-Trigger` or JS events |
| `HX-Trigger-After-Swap` | Trigger events after the swap phase. **[htmx 4 removed]** — use `HX-Trigger` or JS events |

### Triggering Client Events from Server

```
HX-Trigger: myEvent
```

With data:

```
HX-Trigger: {"showMessage": {"level": "info", "message": "Item saved!"}}
```

Multiple events:

```
HX-Trigger: {"event1": null, "event2": {"key": "value"}}
```

### No POST/Redirect/GET Required

HTMX does not need the traditional POST/Redirect/GET pattern. After a successful POST, the server can return new HTML directly — no 302 redirect needed.

### 204 No Content

A `204 No Content` response performs no swap. Useful for fire-and-forget requests.

### Error Responses

> **[htmx 4 change]** All HTTP responses now swap by default — only 204 and 304 are excluded. Design error responses as valid swap content, or use `hx-status` to control per-code behavior. Restore v2 behavior with `htmx.config.noSwap = [204, 304, '4xx', '5xx']`.

- Non-2xx responses (except 204) trigger the `htmx:responseError` event
- Network failures trigger the `htmx:sendError` event

> **Important:** Response headers (`HX-Trigger`, `HX-Retarget`, etc.) are **NOT processed** on 3xx redirect responses. Use alternative status codes when returning htmx-specific headers.

### Response Handling Configuration

> **[htmx 4 removed]** `htmx.config.responseHandling` is removed. Use the `hx-status` attribute for per-element status code handling. See `references/attributes.md`.

Override default behavior per status code using `htmx.config.responseHandling`:

```javascript
htmx.config.responseHandling = [
    { code: "204", swap: false },
    { code: "[23]..", swap: true },
    { code: "422", swap: true, error: true },
    { code: "[45]..", swap: false, error: true },
    // Custom: swap 404 responses into a specific target
    { code: "404", swap: true, target: "#error-container" }
];
```

Fields per entry:

| Field | Type | Purpose |
|---|---|---|
| `code` | String (regex) | Status code pattern to match |
| `swap` | Boolean | Whether to swap content into DOM |
| `error` | Boolean | Whether to treat as error (fires error events) |
| `ignoreTitle` | Boolean | Skip `<title>` updates |
| `select` | String | CSS selector for response content |
| `target` | String | Alternative target selector |
| `swapOverride` | String | Custom swap strategy |

## Parameters & Values

### What Gets Sent

| Scenario | Included Values |
|---|---|
| Input/select/textarea triggers request | Its own `name`/`value` pair |
| Element inside a form (non-GET) | All form input values |
| `hx-include` specified | Values from the targeted elements |
| `hx-vals` specified | Additional JSON key-value pairs |
| `hx-params` specified | Filtered subset of values |

### Parameter Encoding

| Method | Encoding |
|---|---|
| GET, DELETE | URL query string parameters. **[htmx 4 change]** DELETE no longer includes enclosing form data (matches GET). Use `hx-include="closest form"` if needed |
| POST, PUT, PATCH | URL-encoded form body (`application/x-www-form-urlencoded`) |
| With `hx-encoding="multipart/form-data"` | Multipart form body |

### Programmatic Parameter Modification

Use the `htmx:configRequest` event:

```html
<div hx-get="/data" hx-on:htmx:config-request="event.detail.parameters.extra = 'value'">
    Load
</div>
```

Or globally:

```javascript
document.body.addEventListener('htmx:configRequest', function(event) {
    event.detail.parameters['auth'] = getToken();
    event.detail.headers['X-Custom'] = 'value';
});
```

## File Upload

Set `hx-encoding="multipart/form-data"` to enable file uploads:

```html
<form hx-post="/upload" hx-encoding="multipart/form-data">
    <input type="file" name="document" />
    <button type="submit">Upload</button>
</form>
```

Track upload progress via the `htmx:xhr:progress` event:

> **[htmx 4 removed]** `htmx:xhr:progress` is removed (htmx 4 uses `fetch()` instead of XHR). File upload progress tracking requires a different approach in v4.

```html
<form hx-post="/upload" hx-encoding="multipart/form-data"
      hx-on:htmx:xhr:progress="updateProgress(event)">
    <input type="file" name="document" />
    <progress id="progress" value="0" max="100"></progress>
    <button type="submit">Upload</button>
</form>

<script>
    function updateProgress(event) {
        var progress = event.detail.loaded / event.detail.total * 100;
        document.getElementById('progress').value = progress;
    }
</script>
```

## CSRF Protection

### Include Token via hx-headers

Place on `<body>` or a high-level element to inherit to all requests:

```html
<body hx-headers='{"X-CSRF-TOKEN": "{{ csrf_token }}"}'>
    ...
</body>
```

### Include Token via htmx:configRequest

```javascript
document.body.addEventListener('htmx:configRequest', function(event) {
    event.detail.headers['X-CSRF-TOKEN'] = document.querySelector('meta[name="csrf-token"]').content;
});
```

### Framework Patterns

Most server frameworks auto-insert CSRF tokens into `<form>` elements. For non-form HTMX requests, use one of the above approaches.

## CORS

For cross-origin requests, configure the server to allow HTMX headers:

```
Access-Control-Allow-Headers: HX-Request, HX-Trigger, HX-Trigger-Name, HX-Target, HX-Current-URL, HX-Boosted, HX-Prompt
Access-Control-Expose-Headers: HX-Location, HX-Push-Url, HX-Redirect, HX-Refresh, HX-Replace-Url, HX-Reswap, HX-Retarget, HX-Reselect, HX-Trigger, HX-Trigger-After-Settle, HX-Trigger-After-Swap

> **[htmx 4 change]** Update CORS headers: `HX-Trigger` request header → `HX-Source`. Remove `HX-Trigger-Name`, `HX-Prompt` from allow-headers. Remove `HX-Trigger-After-Settle`, `HX-Trigger-After-Swap` from expose-headers.
```

**Note:** `htmx.config.selfRequestsOnly = true` (default) restricts requests to the same domain. Set to `false` for cross-origin requests, or use the `htmx:validateUrl` event for fine-grained control.

## Caching

HTMX respects standard HTTP caching headers:

| Header | Behavior |
|---|---|
| `Last-Modified` | HTMX sends `If-Modified-Since` on subsequent requests |
| `ETag` | HTMX sends `If-None-Match` on subsequent requests |
| `Cache-Control` | Standard browser caching applies |

### Preventing Mixed Cache

Use `Vary: HX-Request` to prevent caching HTMX partial responses as full pages:

```
Vary: HX-Request
```

### Cache Busting

```javascript
htmx.config.getCacheBusterParam = true;
```

Appends a unique `org.htmx.cache-buster` query parameter to GET requests.

## Security

### Core Rule

**Always escape all user-supplied content server-side.** Whitelist allowed HTML tags/attributes rather than blacklisting.

### hx-disable

> **[htmx 4 change]** Renamed to `hx-ignore` in v4.

Prevents HTMX processing on an element and all children. Use to protect user-content areas:

```html
<div hx-disable>
    {{ user_generated_content }}
</div>
```

### Content Security Policy (CSP)

```html
<meta http-equiv="Content-Security-Policy" content="default-src 'self';">
```

### Configuration for Hardening

> **[htmx 4 change]** `selfRequestsOnly` → `mode` (`'same-origin'` default). `allowEval`, `allowScriptTags`, `historyCacheSize` are removed in v4.

```javascript
// Restrict to same-origin requests only (default: true)
htmx.config.selfRequestsOnly = true;

// Disable eval-dependent features
htmx.config.allowEval = false;

// Disable script tag processing in responses
htmx.config.allowScriptTags = false;

// Disable history cache (no localStorage usage)
htmx.config.historyCacheSize = 0;
```

### URL Validation

Use `htmx:validateUrl` to inspect and block requests:

```javascript
document.body.addEventListener('htmx:validateUrl', function(event) {
    if (!event.detail.sameHost) {
        event.preventDefault(); // Block cross-origin request
    }
});
```

## Validation

HTMX integrates with the HTML5 Validation API. If a form element is invalid, the request is blocked.

```html
<form hx-post="/submit">
    <input name="email" type="email" required />
    <button type="submit">Submit</button>
</form>
```

### Validation on Non-Form Elements

```html
<input hx-get="/check" hx-validate="true" required pattern="[a-z]+" />
```

### Validation Events

> **[htmx 4 removed]** Validation events are removed. Use native HTML form validation.

| Event | When |
|---|---|
| `htmx:validation:validate` | Before `checkValidity()` is called |
| `htmx:validation:failed` | After a validation failure |
| `htmx:validation:halted` | When a request is halted due to validation |

### Browser Validation Alerts

```javascript
htmx.config.reportValidityOfForms = true; // default: false
```

## Confirmation & Prompts

### Simple Confirm

```html
<button hx-delete="/item/1" hx-confirm="Delete this item?">Delete</button>
```

### Custom Confirm Dialog

Use the `htmx:confirm` event for non-native dialogs:

```javascript
document.body.addEventListener('htmx:confirm', function(event) {
    event.preventDefault(); // Halt the request

    showCustomDialog(event.detail.question).then(function(confirmed) {
        if (confirmed) {
            event.detail.issueRequest(); // Continue the request
        }
    });
});
```

### Prompt

> **[htmx 4 change]** `hx-prompt` is removed from core but restored by the official `hx-prompt` extension (beta5) — load `dist/ext/hx-prompt.js` and the attribute and `HX-Prompt` header work as in v2. Without the extension, use `hx-confirm` with `js:` prefix or set the header via `hx-on::config:request="ctx.request.headers['HX-Prompt'] = prompt('...') ?? event.preventDefault()"`.

```html
<button hx-post="/rename" hx-prompt="Enter new name:">Rename</button>
```

The user's response is available in the `HX-Prompt` request header.

## Sources

[^1]: htmx documentation — AJAX, triggers, parameters, and request lifecycle. <https://htmx.org/docs/#ajax>

[^2]: htmx reference — request and response headers. <https://htmx.org/reference/#headers>

[^3]: htmx documentation — CSRF, security, and request configuration. <https://htmx.org/docs/#security>

[^4]: htmx 4 migration guide — header renames (`HX-Trigger` → `HX-Source`), removed headers (`HX-Trigger-Name`, `HX-Trigger-After-Settle`, `HX-Trigger-After-Swap`; `HX-Prompt` restored via the `hx-prompt` extension in beta5), new headers (`HX-Request-Type`, `Accept`), fetch replacing XHR, default timeout change. <https://four.htmx.org/docs/get-started/migration/>
