---
description: HTMX swap strategies, targets, OOB swaps, morphing, and transitions.
globs: "*.html"
---

# HTMX Swapping


How HTMX places response content into the DOM.

## Contents

- Swap Strategies
- Swap Modifiers
- Out-of-Band (OOB) Swaps
- Morphing
- CSS Transitions
- View Transitions
- Preserving Content

## Swap Strategies

The `hx-swap` attribute controls how the response is inserted relative to the target element.

| Strategy | Behavior |
|---|---|
| `innerHTML` | **Default.** Replace the target's inner content |
| `outerHTML` | Replace the entire target element (including itself) |
| `textContent` | Replace the target's text content, without parsing HTML |
| `beforebegin` | Insert before the target element (as a sibling) |
| `afterbegin` | Insert inside the target, before its first child |
| `beforeend` | Insert inside the target, after its last child |
| `afterend` | Insert after the target element (as a sibling) |
| `delete` | Delete the target element, no content swap |
| `none` | No content swap. OOB swaps and response headers still process |
| `innerMorph` | **[htmx 4]** Morph target's children using Idiomorph (preserves state) |
| `outerMorph` | **[htmx 4]** Morph entire target element using Idiomorph |

> **[htmx 4]** Short aliases: `before` = `beforebegin`, `after` = `afterend`, `prepend` = `afterbegin`, `append` = `beforeend`. Both old and new names work.

```html
<!-- Replace inner content (default) -->
<div hx-get="/data" hx-swap="innerHTML">Loading...</div>

<!-- Replace the whole element -->
<div hx-get="/data" hx-swap="outerHTML">Replace me entirely</div>

<!-- Append to a list -->
<button hx-get="/more-items" hx-target="#list" hx-swap="beforeend">
    Load More
</button>

<!-- Delete on success -->
<button hx-delete="/item/1" hx-target="closest tr" hx-swap="delete">
    Remove
</button>

<!-- Fire-and-forget (no DOM update) -->
<button hx-post="/track" hx-swap="none">Track Click</button>
```

**Note:** Using `outerHTML` on `<body>` is automatically converted to `innerHTML` due to DOM limitations. `hx-swap` is inherited.

## Swap Modifiers

Append modifiers to `hx-swap` separated by spaces. Each modifier uses `key:value` syntax.

```html
<div hx-get="/data" hx-swap="innerHTML swap:300ms settle:100ms scroll:top">
    Content
</div>
```

| Modifier | Effect | Default |
|---|---|---|
| `swap:<time>` | Delay between old content removal and new content insertion | `0` |
| `settle:<time>` | Delay between new content insertion and settle phase | `20ms` |
| `transition:true` | Use the View Transitions API for this swap | `false` |
| `ignoreTitle:true` | Don't update `document.title` from response `<title>` | `false` |
| `scroll:top` | Scroll target element to its top after swap | — |
| `scroll:bottom` | Scroll target element to its bottom after swap | — |
| `scroll:<selector>:top` | Scroll specified element to top | — |
| `scroll:<selector>:bottom` | Scroll specified element to bottom | — |
| `show:top` | Scroll viewport so target's top is visible | — |
| `show:bottom` | Scroll viewport so target's bottom is visible | — |
| `show:<selector>:top` | Scroll viewport so specified element's top is visible. **[htmx 4 change]** Use `show:top showTarget:<selector>` instead | — |
| `show:<selector>:bottom` | Scroll viewport so specified element's bottom is visible. **[htmx 4 change]** Use `show:bottom showTarget:<selector>` instead | — |
| `show:window:top` | Scroll viewport to the top of the window | — |
| `show:window:bottom` | Scroll viewport to the bottom of the window | — |
| `show:none` | Disable all scroll-into-view behavior | — |
| `focus-scroll:true` | Auto-scroll to focused element after swap | `false` |
| `swapEmpty:<bool>` | **[htmx 4]** Whether an empty response body still performs the main swap (beta5). `swapEmpty:false` leaves the target unchanged; `swapEmpty:true` (or bare `swapEmpty`) forces the swap, clearing the target. Default from `htmx.config.defaultSwapEmpty`; when unset, empty responses swap except `<hx-partial>`-only responses. SSE messages default to `swapEmpty:false` | — |

### Scroll Examples

```html
<!-- Scroll the target to top after loading -->
<div hx-get="/data" hx-swap="innerHTML scroll:top">Content</div>

<!-- Scroll viewport so #results top is visible -->
<button hx-get="/search" hx-target="#results" hx-swap="innerHTML show:#results:top">
    Search
</button>

<!-- Infinite scroll: append and show last item -->
<div hx-get="/page/2" hx-swap="beforeend show:bottom">...</div>
```

## Out-of-Band (OOB) Swaps

OOB swaps update elements **outside** the target using their `id` attributes. This enables updating multiple page regions from a single response.

> **[htmx 4 change]** OOB swap order changed: main content swaps first, then OOB and `<hx-partial>` elements (in document order). In v2, OOB swapped before main content.

> **[htmx 4]** New `<hx-partial>` element provides cleaner multi-target responses. Each `<hx-partial>` specifies its own `hx-target` and `hx-swap`:
> ```html
> <hx-partial hx-target="#messages" hx-swap="beforeend">
>     <div>New message</div>
> </hx-partial>
> <hx-partial hx-target="#count">
>     <span>5</span>
> </hx-partial>
> ```

### hx-swap-oob on Response Elements

The server response includes elements marked with `hx-swap-oob`:

```html
<!-- Server response -->
<!-- This goes into the main target normally -->
<div>Main content here</div>

<!-- This swaps into #notification by its id, out of band -->
<div id="notification" hx-swap-oob="true">
    Item saved successfully!
</div>

<!-- OOB with specific strategy -->
<div id="item-count" hx-swap-oob="innerHTML">42</div>

<!-- OOB append -->
<tr id="table-body" hx-swap-oob="beforeend">
    <td>New row</td>
</tr>
```

### hx-swap-oob Values

| Value | Behavior |
|---|---|
| `true` | Replace the entire element by matching `id` (uses `outerHTML`) |
| `innerHTML` | Replace inner content of element with matching `id` |
| `outerHTML` | Replace entire element with matching `id` |
| `beforebegin` | Insert before element with matching `id` |
| `afterbegin` | Prepend inside element with matching `id` |
| `beforeend` | Append inside element with matching `id` |
| `afterend` | Insert after element with matching `id` |
| `delete` | Delete element with matching `id` |
| `none` | Do nothing (suppresses OOB) |

### hx-select-oob on Requesting Elements

Pick specific elements from the response for OOB swapping:

```html
<!-- The button: swap #results into target, and OOB swap #count and #status -->
<button hx-get="/data"
        hx-target="#results"
        hx-select="#results"
        hx-select-oob="#count:innerHTML, #status:outerHTML">
    Load
</button>
```

Format: `#id:strategy, #id:strategy, ...`

### OOB Pattern: Update Multiple Regions

Server returns one response that updates several page areas:

```html
<!-- Response from server -->
<div id="main-content">
    <h2>Updated Content</h2>
    <p>The primary content area.</p>
</div>

<div id="sidebar-count" hx-swap-oob="innerHTML">5 items</div>
<div id="notification" hx-swap-oob="afterbegin">
    <div class="alert">Content updated!</div>
</div>
<div id="last-updated" hx-swap-oob="innerHTML">Just now</div>
```

## Morphing

Morphing merges new HTML into existing DOM, preserving focus, scroll position, and element state. Requires an extension.

> **[htmx 4 change]** Idiomorph is built into core — no extension needed. Use `hx-swap="innerMorph"` or `hx-swap="outerMorph"` directly. New config keys `morphIgnore`, `morphSkip`, `morphSkipChildren` control morph behavior via CSS selectors. As of beta5, `morphSkip` defaults to `[hx-morph-skip]` and `morphSkipChildren` to `[hx-morph-skip-children]` — add `hx-morph-skip` to any element in server templates to freeze it entirely during a morph, or `hx-morph-skip-children` to update its attributes while leaving its children untouched (declarative, server-driven morph freezing; no config needed).

### Idiomorph (Recommended)

HTMX's own morphing algorithm. Matches elements by `id` and structure.

```html
<head>
    <script src="https://unpkg.com/idiomorph@0.7.3/dist/idiomorph-ext.min.js"></script>
</head>
<body hx-ext="morph">
    <!-- Morph swap instead of replacing -->
    <div hx-get="/data" hx-swap="morph">Content</div>

    <!-- Morph just the inner content -->
    <div hx-get="/data" hx-swap="morph:innerHTML">Content</div>

    <!-- Morph the outer element -->
    <div hx-get="/data" hx-swap="morph:outerHTML">Content</div>
</body>
```

### Other Morph Extensions

| Extension | Package | Notes |
|---|---|---|
| Idiomorph | `idiomorph` | HTMX team's recommended morph engine |
| Morphdom Swap | `htmx-ext-morphdom-swap` | Based on the morphdom library |
| Alpine-morph | `htmx-ext-alpine-morph` | Integrates with Alpine.js morph plugin |

### When to Use Morphing

- Forms with focus/cursor position that must be preserved
- Lists with animations where elements reorder
- Complex UIs where replacing breaks third-party widget state
- Any case where full replacement causes visual flicker

## CSS Transitions

HTMX works with CSS transitions when:

1. The element has a **stable `id`** across requests
2. The new content changes a CSS class or property
3. CSS defines a transition for that property

### How It Works

During swap, HTMX:
1. Copies old element attributes to the new element
2. Inserts new content with old values
3. After one frame, applies new values
4. CSS transition animates between old and new

### Fade-Out on Swap

```html
<style>
    .fade-me-out.htmx-swapping {
        opacity: 0;
        transition: opacity 1s ease-out;
    }
</style>

<!-- swap:1s gives CSS transition time to complete before removal -->
<button class="fade-me-out"
        hx-delete="/fade_out_demo"
        hx-swap="outerHTML swap:1s">
    Fade Me Out
</button>
```

### Fade-In on Addition

```html
<style>
    #fade-me-in.htmx-added {
        opacity: 0;
    }
    #fade-me-in {
        opacity: 1;
        transition: opacity 1s ease-out;
    }
</style>

<button id="fade-me-in"
        hx-post="/fade_in_demo"
        hx-swap="outerHTML settle:1s">
    Fade Me In
</button>
```

### Request In-Flight Animation

```html
<style>
    form.htmx-request {
        opacity: .5;
        transition: opacity 300ms linear;
    }
</style>

<form hx-post="/name" hx-swap="outerHTML">
    <label>Name:</label><input name="name" />
    <button type="submit">Submit</button>
</form>
```

### Transition Classes

| Class | Applied When | Removed When |
|---|---|---|
| `htmx-request` | Request starts | Request ends |
| `htmx-swapping` | Before swap | After swap |
| `htmx-added` | After swap (on new elements) | After settle |
| `htmx-settling` | After swap | After settle |

## View Transitions

The View Transitions API provides browser-native animated transitions.

### Enable Per-Element

```html
<div hx-get="/page" hx-swap="innerHTML transition:true">Content</div>
```

### Enable Globally

> **[htmx 4 change]** Config key renamed from `globalViewTransitions` to `transitions`.

```javascript
htmx.config.globalViewTransitions = true;
```

### Styling View Transitions

```css
/* Customize the transition animation */
::view-transition-old(root) {
    animation: fade-out 0.2s ease-out;
}
::view-transition-new(root) {
    animation: fade-in 0.2s ease-in;
}

/* Named transitions for specific elements */
#content {
    view-transition-name: content;
}
::view-transition-old(content) {
    animation: slide-out 0.3s;
}
::view-transition-new(content) {
    animation: slide-in 0.3s;
}
```

### Preventing View Transitions

Cancel via the `htmx:beforeTransition` event:

```javascript
document.addEventListener('htmx:beforeTransition', function(event) {
    // Cancel transition based on condition
    if (shouldSkipTransition) {
        event.preventDefault();
    }
});
```

## Preserving Content

### hx-preserve

Keep an element completely unchanged across swaps. The element must have an `id`.

```html
<div hx-get="/page" hx-select="#content" hx-target="#content">
    <div id="content">
        <video id="player" hx-preserve>
            <source src="video.mp4" />
        </video>
        <p>Other content that will update</p>
    </div>
</div>
```

Use cases:
- Video/audio players that should keep playing
- Iframes that should maintain state
- Complex third-party widgets

### HTML Element Encapsulation for OOB

Browsers enforce strict HTML structure. OOB swap elements may need wrapping:

| Element Type | Must Be Wrapped In |
|---|---|
| `<tr>`, `<td>`, `<th>` | `<tbody>`, `<table>`, or `<template>` |
| `<li>` | `<ul>`, `<ol>`, `<div>`, `<span>`, or `<template>` |
| `<p>` | `<div>` or `<span>` |
| SVG elements | `<template>` + `<svg>` (preserve XML namespace) |

```html
<template>
    <tr id="row-1" hx-swap-oob="true">
        <td>Updated cell</td>
    </tr>
</template>
```

## Sources

[^1]: htmx documentation — swapping strategies, targets, and OOB swaps. <https://htmx.org/docs/#swapping>

[^2]: htmx documentation — CSS transitions. <https://htmx.org/docs/#css_transitions>

[^3]: htmx documentation — View Transitions API. <https://htmx.org/docs/#view-transitions>

[^4]: htmx reference — `hx-swap` attribute values. <https://htmx.org/reference/#attributes>

[^5]: htmx 4 migration guide — `innerMorph`/`outerMorph` swap strategies, short aliases (`before`, `after`, `prepend`, `append`), `<hx-partial>` element, OOB swap order change (main first, then OOB), scroll modifier syntax change. <https://four.htmx.org/docs/get-started/migration/>
