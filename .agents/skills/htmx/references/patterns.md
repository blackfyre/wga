---
description: Common HTMX UI patterns and implementation examples.
globs: "*.html"
---

# HTMX Patterns


Common UI patterns with HTMX implementation examples.

## Contents

- Click To Edit
- Inline Validation
- Active Search
- Infinite Scroll
- Click to Load
- Lazy Loading
- Progress Bar
- Cascading Selects
- Bulk Update
- Delete Row
- Edit Row
- Tabs (HATEOAS)
- Tabs (JavaScript)
- Dialogs & Modals
- File Upload with Progress
- Keyboard Shortcuts
- Updating Other Content
- Async Authentication
- Polling
- Drag & Drop / Sortable

## Click To Edit

Inline editing of a data object. Click a view to swap in an edit form; submit to swap back.

```html
<!-- View mode (server returns this) -->
<div hx-target="this" hx-swap="outerHTML">
    <p><strong>Name:</strong> John Doe</p>
    <p><strong>Email:</strong> john@example.com</p>
    <button hx-get="/contact/1/edit">Edit</button>
</div>
```

```html
<!-- Edit mode (server returns on GET /contact/1/edit) -->
<form hx-put="/contact/1" hx-target="this" hx-swap="outerHTML">
    <label>Name: <input name="name" value="John Doe" /></label>
    <label>Email: <input name="email" value="john@example.com" /></label>
    <button type="submit">Save</button>
    <button hx-get="/contact/1">Cancel</button>
</form>
```

**Server:** `GET /contact/1/edit` returns the form. `PUT /contact/1` saves and returns the view.

## Inline Validation

Validate fields as the user fills them out. The **wrapping div** is the target — server replaces the entire container with validation styling.

```html
<form hx-post="/register">
    <div hx-target="this" hx-swap="outerHTML">
        <label>Email</label>
        <input name="email" hx-post="/validate/email" hx-indicator="#ind" />
        <img id="ind" src="/img/bars.svg" class="htmx-indicator" />
    </div>

    <div>
        <label>Username</label>
        <input name="username" type="text" />
    </div>

    <button type="submit">Register</button>
</form>
```

**Server returns on validation error (replaces the entire div):**

```html
<div hx-target="this" hx-swap="outerHTML" class="error">
    <label>Email</label>
    <input name="email" hx-post="/validate/email" hx-indicator="#ind"
           value="taken@example.com" />
    <img id="ind" src="/img/bars.svg" class="htmx-indicator" />
    <div class="error-message">That email is already taken.</div>
</div>
```

```css
.error input { box-shadow: 0 0 3px #CC0000; }
.valid input { box-shadow: 0 0 3px #36cc00; }
.error-message { color: red; }
```

**Key points:**
- `hx-target="this"` on the wrapping div — the input's POST replaces the whole div
- Server adds `class="error"` or `class="valid"` to the wrapper
- Default trigger for `<input>` is `change` — no explicit `hx-trigger` needed

## Active Search

Live search with debounced input, Enter key shortcut, and initial load.

```html
<h3>
    Search Contacts
    <span class="htmx-indicator">
        <img src="/img/bars.svg" /> Searching...
    </span>
</h3>
<input class="form-control" type="search"
       name="search" placeholder="Begin Typing To Search Users..."
       hx-post="/search"
       hx-trigger="input changed delay:500ms, keyup[key=='Enter'], load"
       hx-target="#search-results"
       hx-indicator=".htmx-indicator" />

<table class="table">
    <thead>
    <tr>
        <th>First Name</th>
        <th>Last Name</th>
        <th>Email</th>
    </tr>
    </thead>
    <tbody id="search-results">
    </tbody>
</table>
```

**Key points:**
- `hx-post` (not GET) sends the search value in the request body
- `input changed delay:500ms` — debounces, skips if value unchanged
- `keyup[key=='Enter']` — event filter for immediate search on Enter
- `load` — fires on page load to show initial results
- `changed` modifier prevents duplicate requests from arrow keys or non-character keys

## Infinite Scroll

Load more content as the user scrolls.

```html
<table>
    <thead>
        <tr><th>Name</th><th>Email</th><th>ID</th></tr>
    </thead>
    <tbody>
        <tr><td>Agent Smith</td><td>void1@null.org</td><td>1</td></tr>
        <tr><td>Agent Smith</td><td>void2@null.org</td><td>2</td></tr>
        <!-- ... initial items ... -->

        <!-- Last row triggers next page load when scrolled into view -->
        <tr hx-get="/contacts/?page=2"
            hx-trigger="revealed"
            hx-swap="afterend">
            <td>Agent Smith</td><td>void10@null.org</td><td>10</td>
        </tr>
    </tbody>
</table>
```

**Server returns:** The next batch of `<tr>` elements appended **after** the trigger row. The last row of each response carries the `hx-trigger="revealed"` for the next page. When no more pages, return rows without a trigger.

**Key:** Use `hx-swap="afterend"` (not `outerHTML`) so the trigger row stays as a regular data row and new rows append after it.

### Intersection Observer Variant

```html
<div hx-get="/items?page=2"
     hx-trigger="intersect once threshold:0.5"
     hx-swap="afterend">
    Loading more...
</div>
```

> **Caveat:** If your scrollable container uses CSS `overflow-y: scroll`, you **must** use `intersect once` instead of `revealed`. The `revealed` trigger does not work correctly inside custom scrollable containers.

## Click to Load

Manual "Load More" button that fetches the next page of results.

```html
<table>
    <thead><tr><th>Name</th><th>Email</th><th>ID</th></tr></thead>
    <tbody>
        <tr><td>Agent Smith</td><td>void1@null.org</td><td>1</td></tr>
        <!-- ... initial rows ... -->

        <!-- Load more button row (replaces itself) -->
        <tr id="replaceMe">
            <td colspan="3">
                <button class="btn primary" hx-get="/contacts/?page=2"
                        hx-target="#replaceMe"
                        hx-swap="outerHTML">
                    Load More Agents...
                    <img class="htmx-indicator" src="/img/bars.svg" />
                </button>
            </td>
        </tr>
    </tbody>
</table>
```

**Server returns:** New data rows plus a new "Load More" row pointing to `?page=3`. The button replaces itself with new rows + a new button, creating a self-perpetuating chain.

## Lazy Loading

Defer loading content until the element is in the DOM.

```html
<!-- Loads immediately when rendered -->
<div hx-get="/dashboard/stats" hx-trigger="load">
    <span class="htmx-indicator">Loading stats...</span>
</div>

<!-- Loads when scrolled into viewport -->
<div hx-get="/dashboard/chart" hx-trigger="revealed">
    <span class="htmx-indicator">Loading chart...</span>
</div>
```

## Progress Bar

Server-driven progress updates using polling and event-based completion.

```html
<!-- Start the job -->
<div hx-target="this" hx-swap="outerHTML">
    <h3>Start Progress</h3>
    <button class="btn primary" hx-post="/start">
        Start Job
    </button>
</div>
```

**Server returns after POST /start (running state with nested polling):**

```html
<div hx-trigger="done" hx-get="/job" hx-swap="outerHTML" hx-target="this">
    <h3 role="status">Running</h3>

    <div hx-get="/job/progress"
         hx-trigger="every 600ms"
         hx-target="this"
         hx-swap="innerHTML">
        <div class="progress" role="progressbar"
             aria-valuemin="0" aria-valuemax="100" aria-valuenow="0">
            <div id="pb" class="progress-bar" style="width:0%"></div>
        </div>
    </div>
</div>
```

**Server returns on each GET /job/progress (while in progress):**

```html
<div class="progress" role="progressbar"
     aria-valuemin="0" aria-valuemax="100" aria-valuenow="45">
    <div id="pb" class="progress-bar" style="width:45%"></div>
</div>
```

**Server signals completion via response header:**

```
HX-Trigger: done
```

This fires the `done` event on the outer div, which triggers `hx-get="/job"` to fetch the complete state. The complete state sets `hx-trigger="none"` to stop polling.

**Key techniques:**
- `every 600ms` for polling (not load polling)
- `HX-Trigger: done` response header signals completion
- Outer div listens for custom `done` event
- Stable `id="pb"` enables CSS `transition: width .6s ease` for smooth bar updates
- `hx-trigger="none"` stops polling in the complete state

## Cascading Selects

Second select populates based on first select's value.

```html
<div>
    <label>Make</label>
    <select name="make" hx-get="/models" hx-target="#models"
            hx-indicator=".htmx-indicator">
        <option value="audi">Audi</option>
        <option value="toyota">Toyota</option>
        <option value="bmw">BMW</option>
    </select>
</div>
<div>
    <label>Model</label>
    <select id="models" name="model">
        <option value="a1">A1</option>
    </select>
    <img class="htmx-indicator" width="20" src="/img/bars.svg" />
</div>
```

**Server returns for GET /models?make=bmw:**

```html
<option value="325i">325i</option>
<option value="325ix">325ix</option>
<option value="X5">X5</option>
```

**Key points:**
- Default trigger for `<select>` is `change` — no explicit `hx-trigger` needed
- The selected value is automatically included as a query parameter
- No `hx-include` needed — `<select>` sends its own value by default

## Bulk Update

Activate/deactivate multiple rows. The table is **not re-rendered** — only a toast message is returned.

```html
<form id="checked-contacts"
      hx-post="/users"
      hx-swap="innerHTML settle:3s"
      hx-target="#toast">
    <table>
        <thead>
        <tr><th>Name</th><th>Email</th><th>Active</th></tr>
        </thead>
        <tbody>
            <tr>
                <td>Joe Smith</td>
                <td>joe@smith.org</td>
                <td><input type="checkbox" name="active:joe@smith.org" checked /></td>
            </tr>
            <tr>
                <td>Kim Yee</td>
                <td>kim@yee.org</td>
                <td><input type="checkbox" name="active:kim@yee.org" /></td>
            </tr>
        </tbody>
    </table>
    <input type="submit" value="Bulk Update" class="btn primary" />
    <output id="toast"></output>
</form>
```

Toast animation CSS:

```css
#toast.htmx-settling {
    opacity: 100;
}
#toast {
    background: #E1F0DA;
    opacity: 0;
    transition: opacity 3s ease-out;
}
```

**Key points:**
- Checkbox `name="active:email"` convention — server parses the key to determine which records to activate
- Unchecked checkboxes are NOT sent (standard HTML) — server deactivates missing emails
- `settle:3s` creates a toast that appears then fades out over 3 seconds
- `<output>` element is semantically appropriate for form results
- The table is not re-rendered — form inputs manage their own state

## Delete Row

Remove a table row with confirmation and fade-out animation. Use **attribute inheritance** by placing shared attributes on `<tbody>`.

```html
<table class="table">
    <thead>
        <tr><th>Name</th><th>Email</th><th>Status</th><th></th></tr>
    </thead>
    <tbody hx-confirm="Are you sure?"
           hx-target="closest tr"
           hx-swap="outerHTML swap:1s">
        <tr>
            <td>Angie MacDowell</td>
            <td>angie@macdowell.org</td>
            <td>Active</td>
            <td>
                <button class="btn danger" hx-delete="/contact/1">
                    Delete
                </button>
            </td>
        </tr>
    </tbody>
</table>
```

Fade-out animation (target the `td` cells, not the `tr`):

```css
tr.htmx-swapping td {
    opacity: 0;
    transition: opacity 1s ease-out;
}
```

**Key points:**
- `hx-confirm`, `hx-target`, `hx-swap` on `<tbody>` — inherited by all delete buttons
- `swap:1s` delay gives CSS transition time to complete before DOM removal
- Server returns empty response (200 OK) to remove the row

## Edit Row

Inline table row editing with mutual exclusion — only one row editable at a time.

```html
<table>
    <thead>
        <tr><th>Name</th><th>Email</th><th></th></tr>
    </thead>
    <tbody hx-target="closest tr" hx-swap="outerHTML">
        <!-- Read-only row -->
        <tr>
            <td>Joe Smith</td>
            <td>joe@smith.org</td>
            <td>
                <button class="btn danger"
                        hx-get="/contact/0/edit"
                        hx-trigger="edit"
                        onClick="let editing = document.querySelector('.editing')
                                 if(editing) {
                                     Swal.fire({title: 'Already Editing',
                                                showCancelButton: true,
                                                confirmButtonText: 'Yep, Edit This Row!',
                                                text:'You are already editing a row!'})
                                     .then((result) => {
                                          if(result.isConfirmed) {
                                             htmx.trigger(editing, 'cancel')
                                             htmx.trigger(this, 'edit')
                                          }
                                      })
                                 } else {
                                    htmx.trigger(this, 'edit')
                                 }">
                    Edit
                </button>
            </td>
        </tr>
    </tbody>
</table>
```

**Server returns for GET /contact/0/edit (editable row):**

```html
<tr hx-trigger="cancel" class="editing" hx-get="/contact/0">
    <td><input autofocus name="name" value="Joe Smith" /></td>
    <td><input name="email" value="joe@smith.org" /></td>
    <td>
        <button class="btn danger" hx-get="/contact/0">Cancel</button>
        <button class="btn danger" hx-put="/contact/0"
                hx-include="closest tr">Save</button>
    </td>
</tr>
```

**Key points:**
- `hx-trigger="edit"` — custom event, not triggered by click directly
- `onClick` checks for existing `.editing` row and uses SweetAlert2 to confirm
- `htmx.trigger(editing, 'cancel')` sends cancel event to existing edit row
- `hx-include="closest tr"` gathers input values without a `<form>` (table rows cannot contain forms)
- `hx-target="closest tr"` and `hx-swap="outerHTML"` on `<tbody>` — inherited

## Tabs (HATEOAS)

Server-driven tabs using hypermedia.

```html
<div id="tabs">
    <a hx-get="/tabs/details"
       hx-target="#tab-content"
       hx-push-url="true"
       class="active">Details</a>
    <a hx-get="/tabs/settings"
       hx-target="#tab-content"
       hx-push-url="true">Settings</a>
    <a hx-get="/tabs/activity"
       hx-target="#tab-content"
       hx-push-url="true">Activity</a>
</div>

<div id="tab-content">
    <!-- Tab content loaded here -->
</div>
```

**Use `htmx.takeClass`** to toggle the active class:

> **[htmx 4 removed]** `htmx.takeClass()` moved to the `hx-live` extension. In v4 use the `take()` helper inside `hx-on` — `hx-on:click="take('.active', 'a')"` (scope defaults to siblings as of beta5) — or `htmx.live.take(elt, '.active', scope)` from JS, or a plain JS equivalent: `this.parentElement.querySelectorAll('.active').forEach(el => el.classList.remove('active')); this.classList.add('active')`.

```html
<a hx-get="/tabs/details"
   hx-target="#tab-content"
   hx-on:htmx:after-on-load="htmx.takeClass(this, 'active')"
   class="active">Details</a>
```

## Tabs (JavaScript)

Client-side tab switching without server requests.

```html
<div id="tabs" hx-target="#tab-content" hx-swap="innerHTML">
    <button hx-get="/tabs/1" class="active"
            hx-on:htmx:after-on-load="htmx.takeClass(this, 'active')">Tab 1</button>
    <button hx-get="/tabs/2"
            hx-on:htmx:after-on-load="htmx.takeClass(this, 'active')">Tab 2</button>
</div>
<div id="tab-content">Initial content</div>
```

## Dialogs & Modals

### Browser Dialogs

```html
<!-- Confirm -->
<button hx-delete="/item/1" hx-confirm="Are you sure?">Delete</button>

<!-- Prompt -->
<button hx-post="/rename" hx-prompt="New name:">Rename</button>
```

> **[htmx 4 change]** In v4, `hx-prompt` requires the official `hx-prompt` extension (restored in beta5); `hx-confirm` works in core unchanged.

### Custom Modal Dialog

```html
<!-- Trigger: load modal content -->
<button hx-get="/modals/edit-profile"
        hx-target="#modal-container"
        hx-swap="innerHTML">
    Edit Profile
</button>

<div id="modal-container"></div>
```

**Server returns:**

```html
<div class="modal-backdrop" hx-on:click="htmx.remove(this)">
    <div class="modal" hx-on:click="event.stopPropagation()">
        <h2>Edit Profile</h2>
        <form hx-put="/profile"
              hx-target="#profile"
              hx-on:htmx:after-request="htmx.remove(closest('.modal-backdrop'))">
            <input name="name" value="John" />
            <button type="submit">Save</button>
            <button type="button"
                    hx-on:click="htmx.remove(closest('.modal-backdrop'))">
                Cancel
            </button>
        </form>
    </div>
</div>
```

### Using htmx:confirm for Custom Dialogs

```javascript
document.body.addEventListener('htmx:confirm', function(event) {
    if (!event.target.hasAttribute('hx-confirm')) return;

    event.preventDefault();

    showMyCustomDialog(event.detail.question).then(function(confirmed) {
        if (confirmed) {
            event.detail.issueRequest();
        }
    });
});
```

## File Upload with Progress

```html
<form hx-post="/upload"
      hx-encoding="multipart/form-data"
      hx-target="#upload-result"
      hx-indicator="#upload-progress">

    <input type="file" name="file" />
    <button type="submit">Upload</button>

    <div id="upload-progress" class="htmx-indicator">
        <progress id="progress-bar" value="0" max="100"></progress>
    </div>
</form>

<div id="upload-result"></div>

<script>
    htmx.on('htmx:xhr:progress', function(event) {
        var percent = (event.detail.loaded / event.detail.total) * 100;
        document.getElementById('progress-bar').value = percent;
    });
</script>
```

## Keyboard Shortcuts

Trigger HTMX requests from keyboard events.

```html
<!-- Global shortcut: Ctrl+K opens search -->
<div hx-get="/search-modal"
     hx-trigger="keyup[ctrlKey&&key=='k'] from:body"
     hx-target="#modal-container">
</div>

<!-- Element-level shortcut -->
<input hx-get="/search"
       hx-trigger="keyup[key=='Enter']"
       hx-target="#results"
       name="q" />
```

## Updating Other Content

Updating content beyond the primary target using multiple strategies.

### Strategy 1: OOB Swaps

Server returns additional elements with `hx-swap-oob`:

```html
<div id="main-content">Updated main content</div>
<div id="sidebar-count" hx-swap-oob="innerHTML">42</div>
```

### Strategy 2: Server-Triggered Events

Server sends `HX-Trigger` header, client elements listen:

```html
<!-- Server response header: HX-Trigger: itemAdded -->
<div hx-get="/sidebar" hx-trigger="itemAdded from:body">
    Sidebar refreshes when itemAdded fires
</div>
```

### Strategy 3: hx-select-oob

Pick specific elements from response for OOB swap:

```html
<button hx-get="/data"
        hx-target="#main"
        hx-select="#main"
        hx-select-oob="#count, #status">
    Load
</button>
```

### Strategy 4: Path Dependencies (hx-trigger with from)

```html
<table hx-get="/items" hx-trigger="itemChanged from:body">
    <!-- Re-fetches when itemChanged event fires -->
</table>

<form hx-post="/items"
      hx-on:htmx:after-request="htmx.trigger(document.body, 'itemChanged')">
    ...
</form>
```

## Async Authentication

Handle token refresh before requests.

```javascript
document.body.addEventListener('htmx:configRequest', async function(event) {
    var token = getStoredToken();

    if (isExpired(token)) {
        event.preventDefault();
        token = await refreshToken();
        storeToken(token);
        // Re-issue the original request
        htmx.ajax(event.detail.verb, event.detail.path, {
            target: event.detail.target,
            values: event.detail.parameters,
            headers: { 'Authorization': 'Bearer ' + token }
        });
        return;
    }

    event.detail.headers['Authorization'] = 'Bearer ' + token;
});
```

## Polling

### Interval Polling

```html
<!-- Poll every 2 seconds -->
<div hx-get="/status" hx-trigger="every 2s">
    Status: Unknown
</div>
```

**Stop polling:** Server responds with HTTP `286`. Server-side example:

```python
@app.route('/job-status/<job_id>')
def job_status(job_id):
    job = get_job(job_id)
    fragment = render_template('job_status.html', job=job)
    if job.status in ('complete', 'failed'):
        # HTTP 286 tells HTMX to stop polling
        return fragment, 286
    return fragment, 200
```

### Load Polling (Server-Controlled)

```html
<!-- Server controls whether to continue polling -->
<div hx-get="/status" hx-trigger="load delay:1s" hx-swap="outerHTML">
    Checking...
</div>
```

Server returns the same element (with trigger) to continue, or a static element (without trigger) to stop.

### Conditional Polling

```html
<div hx-get="/status"
     hx-trigger="every 2s [document.visibilityState === 'visible']">
    Only polls when tab is visible
</div>
```

## Drag & Drop / Sortable

Integration with Sortable.js for reorderable lists.

```html
<script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"></script>

<form id="sortable-list" hx-post="/reorder" hx-trigger="end">
    <div class="item" data-id="1"><input type="hidden" name="order[]" value="1" />Item 1</div>
    <div class="item" data-id="2"><input type="hidden" name="order[]" value="2" />Item 2</div>
    <div class="item" data-id="3"><input type="hidden" name="order[]" value="3" />Item 3</div>
</form>

<script>
    htmx.onLoad(function(content) {
        var el = content.querySelector('#sortable-list');
        if (el) {
            new Sortable(el, {
                animation: 150,
                onEnd: function() {
                    // Update hidden input values to match new order
                    el.querySelectorAll('.item').forEach(function(item, i) {
                        item.querySelector('input').value = i + 1;
                    });
                    htmx.trigger(el, 'end');
                }
            });
        }
    });
</script>
```

## Sources

[^1]: htmx examples. <https://htmx.org/examples/> — official UI pattern examples (click to edit, bulk update, lazy loading, infinite scroll, active search, progress bar, etc.).

[^2]: htmx documentation. <https://htmx.org/docs/> — attribute behavior and request lifecycle underlying all patterns.
