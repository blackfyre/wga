# Design

## Context

Each affected form is inside its own swap target, so every response replaces the form with `outerHTML`:

| Form | Swap target | Free-text fields | `hx-sync` today |
|---|---|---|---|
| `#artwork-filters` | `#artwork-search` (or the Dual Mode pane) | `q`, `technique`, collection query | yes |
| `#artist-filters` | `#artists` | name (`components.Field`) | no |
| `dual-filters-<key>` | its own pane | `<side>_q` (`components.Field`) | no |
| guestbook search (no id) | `#guestbook` | `q` | no |
| glossary search (no id) | `#glossary` | `q` | no |

The HTMX `delay:` modifier is implemented correctly (HTMX 2.0.10). The problem is that the typed field is rebuilt from the server's echoed value, and the pending timer is lost with the old form. Global search already targets a results region outside its form, so it only takes the new delay.

HTMX `hx-preserve` is decided by the incoming fragment. For each `[hx-preserve]` element in the response, HTMX keeps the existing DOM element with the same id instead. HTMX also sends `HX-Trigger` set to the id of the element that issued the request.

## Goals / Non-Goals

**Goals:**
- Keep the server-rendered form and swap targets as they are. Decide preservation on the server, per request.

**Non-Goals:**
- Moving forms outside their swap targets or switching facet updates to `hx-swap-oob`.
- Range-slider behaviour.
- Client-side JavaScript beyond the single document-level text-search script below.

## Decisions

### Preserve text fields only for requests the form itself issued

A handler renders `hx-preserve` on the originating form's free-text fields only when the request's `HX-Trigger` equals that form's id. All other requests render those fields plainly with server values. That covers resets, sort, view, letter and pane links, full loads, and history-cache misses, which carry no trigger id.

The element that stays on the page is always one first rendered without the attribute, because HTMX keeps the existing element. So history snapshots never contain `hx-preserve`. They do, however, record that element's original `value` attribute; the text-search script below corrects this.

- **Alternative: always render `hx-preserve`.** Rejected. Every swap of the block would keep stale text, including reset and navigation.
- **Alternative: target only the results and update facets out of band.** Rejected for now. It is the cleaner contract but restructures five interactions, and the artworks facets depend on results.

### Stable, unique field ids

Preservation is keyed by id:
- The guestbook search form gets an id, because the header check needs one.
- The glossary form gets an id, and the glossary and guestbook fields get ids.
- The artworks collection query field gets an id.
- Dual Mode ids already include the pane prefix (`dualPrefix(side)`), so they stay unique when both panes render. The form id `dual-filters-<key>` distinguishes panes for the header check.

### Shared field component takes an explicit flag

`dto.Field` gains a boolean that `components.Field` renders as `hx-preserve`. Each handler decides the flag from the header and its own form id, which keeps handlers thin adapters. A small shared helper in `internal/utils` compares `HX-Trigger` with an expected id, so the five handlers don't each reimplement the header check.

### A document-level script completes what preservation cannot

Browser verification showed two gaps that markup cannot close:

- **Pending searches are lost.** The debounce timer belongs to the replaced form. When it fires, HTMX drops the request because the form is no longer in the document (`issueAjaxRequest` checks `bodyContains`). The new form never saw those keystrokes, so text typed while a request was in flight is kept but never searched.
- **History restores stale text.** HTMX snapshots history as `innerHTML`. That records the preserved input's `value` attribute, which still holds its first-render value rather than the typed text.

The affected free-text fields always render a `data-text-search` marker; `dto.Field` gains a flag for it. One idempotent script, initialised once from `bootstrap.ts`, attaches document-level listeners. They survive every swap, and the script never binds to swappable elements:

- On `htmx:beforeRequest`, it records, per requesting form, the value each marked field carries.
- On `htmx:afterSettle` for that form's own request, it sets each still-connected recorded field's `value` attribute to the recorded value. The displayed state and its later history snapshot then agree with its URL. If the live value differs from the recorded one, the field receives a synthetic `input` event, which re-arms the new form's debounce so a follow-up request carries the complete value. Swaps issued by other elements are ignored, because their fields are rendered fresh with server values.

Copying the live value on `htmx:beforeHistorySave` was rejected. HTMX saves the snapshot of the state being left, so the live value belongs to the state being entered, and back would show newer text beside older results.

- **Alternative: target only the results and update facets out of band.** Still rejected, for the same reason as above. It would remove both gaps without script, but only by restructuring five interactions.

### Blur does not start a competing search

The `change` trigger on these forms exists for selects, chips, and checkboxes. On a text field, `change` fires when the field loses focus. Clicking a reset or navigation link then issues a second form request, preserved and racing the link's response, which can push the typed query back into the URL. Each form's `change` trigger therefore gains the event filter `[!target.hasAttribute('data-text-search')]`. Text fields search only through the debounced `input` trigger. HTMX evaluates event filters with `Function`, which the current headers permit, because no Content-Security-Policy is set.

### Trigger configuration

All text-search triggers use a 500 ms `input changed` debounce, alongside each form's filtered `change` trigger. The artists, Dual Mode, and guestbook text triggers switch from `from:` bindings to the form-level filter `input[target.hasAttribute('data-text-search')]`. With a `from:` binding, the new form's `changed` check starts from the preserved field's current value, so it swallows the script's re-announcement. A bubbling trigger does not have this problem, as artworks and glossary already show. Range-slider triggers keep their `from:find` bindings. The glossary form gains `changed`. `hx-sync="this:replace"` is added to the artists, Dual Mode, guestbook search, and glossary forms. The 500 ms value is a starting point. The spec names it, so tuning it later means amending that value in the requirement.

## Risks / Trade-offs

- [The browser lacks the `moveBefore` API, so preservation falls back to `replaceChild` and focus may be lost] → Playwright asserts focus after the swap in Chromium. If focus is lost in the fallback path, record it as a limitation.
- [A reset link is clicked while a form-issued request is in flight; `hx-sync` on the form does not cancel the link's request] → The two responses race. If the stale form response lands last, it preserves the field that the reset has already cleared, so the cleared value survives. Only its results would be stale, which is acceptable for this change.
- [A server-side value normalisation, such as trimming, is no longer reflected in the field while typing] → Acceptable. The next non-form swap or full load shows the normalised value.
- [A synthetic `input` event is dispatched while the visitor is still typing] → Harmless. Their next keystroke restarts the same debounce, so one request still follows the pause.
- [A future Content-Security-Policy forbids `unsafe-eval`] → HTMX disables the event filter, and blur-triggered searches return. Replace the filter with a script-side check if a CSP is introduced.
- [The existing Playwright specs use `fill`, which produces a single input event] → They keep passing. The new tests use sequential typing to exercise the debounce.
