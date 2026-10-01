## Context

Source: Claude Design project "WGA", `CHANGELOG.md` entry of 29 September 2026, and its templ export (`js/wga-search-prefs.ts`, `registerSearchPrefs()`). The reference stores one JSON preference object in `localStorage['wga-aw-prefs']` and a `wga_aw_prefs` cookie, hides `.wga-work-actions` unless `<html data-aw-actions="on">`, keeps sort and view as ordinary links, and redirects a bare `/artworks` visit to the stored sort and view. It names server-side cookie defaults as the preferred alternative to that redirect. The prototype file could not be read in this session, so decisions below rest on the changelog and the existing repository contract.

## Goals / Non-Goals

**Goals:** opt-in result actions without a visible flash, remembered toolbar choices that never rewrite an explicit URL, and sort/view links that keep working without JavaScript.

**Non-Goals:** remembering filters, changing Dual Mode result cards, or moving any other preference into cookies.

## Decisions

### Server-side cookie defaults instead of a client redirect

The artwork search handler reads `wga_aw_prefs` and, only for `GET /artworks` with an empty query string, uses its sort, direction, and view as the request state before the normal parsing, canonicalisation, and rendering run. Every artwork search response also reads the cookie's actions flag to render the toggle's label and pressed state, and full pages render `data-aw-actions` on `<html>` whenever a valid cookie is present.

This fits cleanly because sort and view already flow from one `url.Values` through `buildFilters`; applying the stored values there gives the same canonical URL, hidden form fields, sort links, pagination, and `HX-Push-Url` as an explicit request, so nothing downstream changes. It removes the extra round trip of a redirect and the flash of actions turning off and then on. The cookie value is URL-encoded JSON because raw JSON contains characters that are not valid in a cookie value. An absent, malformed, or out-of-range cookie field is ignored field by field and falls back to the ordinary defaults. Responses add `Vary: Cookie` because their content now depends on it.

Alternative considered: the reference's client-side redirect of bare `/artworks`. Rejected because it costs a second request on every remembered visit and still flashes the actions on full loads.

### What counts as "explicit"

Any query parameter on `/artworks`, and every `/artworks/results` fragment request, uses the URL as-is. This mirrors the reference rule that only a bare visit is redirected: a shared link, a bookmarked filter, pagination, and the filter form (which already carries sort and view as hidden fields) are never reinterpreted. `RESET` and the footer `Artworks` link both resolve to bare `/artworks`, so they keep the remembered choices without special handling.

### Browser module

`resources/js/search-prefs.ts` exports `registerSearchPrefs()`, called synchronously from `app.ts` so `<html data-aw-actions>` is set before the bootstrap chunk loads. It binds document-level capture listeners once, so it survives every HTMX swap: a click on a `[data-wga-aw-pref]` link records the sort, direction, and view from that link's `href` before HTMX issues the request, and a click on `[data-wga-aw-actions]` flips the actions setting. Each write updates `localStorage` and the cookie (one year, `Path=/`, `SameSite=Lax`, `Secure` on HTTPS). On `htmx:load` the module re-syncs toggle labels from the stored state, which covers a visitor whose cookie is blocked but whose `localStorage` works. `localStorage` is the client's source of truth; when it holds preferences the cookie is refreshed from it on each load.

### Preference storage is optional and consent-gated

The remembered choices are not essential, so they sit behind a new CookieConsent category, `preferences`, which is off by default. `search-prefs.ts` runs before the consent library loads, so it reads the CookieConsent record (`cc_cookie`, its `categories` list) directly to decide at start-up; afterwards the library's `onConsent` and `onChange` callbacks call `setSearchPrefsConsent(acceptedCategory("preferences"))`. Without consent the module keeps the choices in memory for the current page, never reads or writes `localStorage` or the cookie, and deletes any copy it finds, so the server sees no cookie and renders defaults. Granting consent saves the current choices; withdrawing it deletes both copies while the page keeps its state. The server-side cookie read is unchanged: with no consent there is no cookie to read.

The privacy policy is database content (`static_pages`, slug `privacy-policy`) and describes cookies generically without listing individual ones, so it is not changed here; the consent notice and preferences dialog carry the specific description.

### No-JavaScript honesty

The actions toggle is a client-only control. It is laid out but `visibility: hidden` until the module marks `<html data-wga-search-prefs>`, so the toolbar does not shift when the module starts and a visitor without JavaScript, or whose bundle failed, is not offered a dead control. The server-rendered `data-aw-actions` cannot serve as that marker because a cookie outlives the script that wrote it. Without JavaScript the actions stay hidden and sort/view remain ordinary links.

### Back to top

The footer link is a plain `href="#top"` fragment link, and the shared header carries `id="top"`. The repository has no global smooth `scroll-behavior`, so the jump uses the browser default and the existing reduced-motion rule; adding site-wide smooth scrolling would change every in-page anchor and is outside this change. The unused `.jump.back-to-top` click handler and its fixed-position rule are removed after confirming no template renders `.jump`.

## Risks / Trade-offs

- [A visitor who disables JavaScript after using it keeps a cookie that can render `data-aw-actions="on"`] → the actions follow the remembered setting, but the toggle stays invisible because only the running module reveals it; sort/view still work.
- [A full load of bare `/artworks` shows remembered state while the address bar stays `/artworks`] → intended: the bare address means "my usual view"; the canonical link and any later HTMX navigation carry the explicit state.
- [Cookie expiry while `localStorage` persists] → the first full load renders defaults and the module refreshes the cookie; the next load is correct.

## Migration Plan

Deploy normally. No data migration; existing visitors start with defaults.
