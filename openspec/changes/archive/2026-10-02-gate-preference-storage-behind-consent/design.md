## Context

On 2 October 2026 the user decided that every remaining browser preference store is optional, so each one must sit behind the existing `preferences` consent category. PR #244 established the pattern in `resources/js/search-prefs.ts`:

- it reads `cc_cookie` live before every read and write;
- it deletes the stored copies when consent is absent or withdrawn;
- it stores the rendered state only when consent is first granted;
- the server checks the same consent in `internal/handlers/artworks/prefs.go`.

## Storage inventory

The inventory comes from a search of `resources/js`, `internal/assets/templ` and the Go handlers for `localStorage`, `sessionStorage`, `document.cookie` and `r.Cookie`, and from the history of #208.

| Item                                     | Kind                                          | Decision                                                                                                                                                                                                                                                                         |
| ---------------------------------------- | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `wga-aw-prefs` / `wga_aw_prefs`          | localStorage + cookie                         | Already gated (#244). It now uses the shared gate.                                                                                                                                                                                                                               |
| `wga-theme`                              | localStorage                                  | Gated. It is read by `appearance.ts` and by the inline head resolver.                                                                                                                                                                                                            |
| `wga-palette`                            | localStorage                                  | Gated. It is read by `appearance.ts` and by the inline head resolver.                                                                                                                                                                                                            |
| `wga-bionic`                             | localStorage                                  | Gated.                                                                                                                                                                                                                                                                           |
| `wga-study-board`                        | localStorage                                  | Gated.                                                                                                                                                                                                                                                                           |
| `wga_theme`, `wga_palette`, `wga_bionic` | legacy JS cookies (one year, set before #208) | Expired on every page load. Nothing has read them since #208, and the server ignores them (`TestFooterIgnoresPaletteAndSchemeCookies`, `TestFooterLeavesBionicReadingNeutralWhenCookieIsPresent`). Browsers may still hold copies, so deleting them honours "delete every copy". |
| `cc_cookie`                              | cookie                                        | Stays. It is the consent record itself, which is strictly necessary to honour the visitor's choice.                                                                                                                                                                              |
| Itinerary session cookie                 | HttpOnly cookie                               | Stays. It is essential: it binds the visitor's itinerary draft and CSRF token to the session they started. It is not a preference.                                                                                                                                               |
| `wga.dynamic-module-recovery`            | sessionStorage                                | Stays. It is a one-shot guard that stops a reload loop when a script chunk fails to load. It is essential and not a preference.                                                                                                                                                  |
| `AppLoggerLevel`                         | localStorage                                  | Stays. It is a developer debugging switch that is written only when someone sets a log level from the console, not a visitor preference.                                                                                                                                         |

No server-read cookie lies behind theme, palette or bionic rendering. #208 removed it, and the no-flash rendering comes from the inline head resolver. That resolver is the only server-rendered piece that needs a gate.

## Decisions

### One shared gate

`resources/js/preference-consent.ts` holds the consent-record parser, `readCookie` and `preferenceStorageAllowed()`, which reads `document.cookie` on every call. It also offers gated `readPreference` and `writePreference`, an always-allowed `removePreference`, and `expireCookie`.

Each store registers a `{ remember, forget }` pair with `registerPreferenceStore`. When a store registers without consent, the gate calls `forget()` to delete any stale copy. The cookie-consent callbacks call `applyPreferenceConsent(acceptedCategory("preferences"))`:

- **Withdrawal:** the gate calls every store's `forget()`.
- **Grant:** the gate calls every store's `remember()` only if this page did not already know consent to be granted. The library also confirms existing consent on every page load, and that confirmation must not overwrite remembered state with what an explicit URL rendered.

The gate takes the initial "already granted" state when the first store registers. `search-prefs.ts` registers synchronously from `app.ts`, before the consent library runs.

The gate reads consent live, so the stores follow a grant or withdrawal made in another tab. A withdrawal in tab B deletes the shared localStorage copies, and tab A's later writes are refused. After a grant in tab B, tab A's next change is stored.

Alternative considered: a copy of the gate in each module. Rejected, because it repeats the parser and the transition rule four times.

### What "the rendered state" means per store

- **Search toolbar:** unchanged. It stores the sort, direction and view rendered on the results, and the `data-aw-actions` value.
- **Palette:** the `<html data-palette>` value.
- **Scheme:** the explicit scheme chosen on this page, if any. `<html data-theme>` cannot be used. A dark-only palette forces it to `dark`, and storing an OS-derived value would stop the scheme following `prefers-color-scheme`. If no scheme was chosen, nothing is stored and the scheme keeps following the OS.
- **Bionic reading:** `<html data-bionic-reading>`, stored as `on` or `off`.
- **Study Board:** the IDs the page currently holds: the URL board on `/study-board`, otherwise the in-memory board.

### Head resolver

The inline resolver in `internal/assets/templ/dto/preferences.go` parses `cc_cookie` (URL-decoded JSON, `categories` containing `preferences`) and reads `wga-palette` and `wga-theme` only when it passes. Otherwise the resolver uses `bone` and `prefers-color-scheme`. It does not delete anything; deletion is left to `appearance.ts`, so a stale copy never shapes the first paint.

### Explicit Study Board URLs

`/study-board?board=…` renders and holds that board for the visit but no longer writes it to storage on load. With consent, the remembered board therefore survives opening someone's shared link. Adding, removing, reordering or clearing works is an explicit edit, so it still writes, as a sort-link click does for the search toolbar. A bare `/study-board` restores the remembered board when consent exists. Without consent, it restores the board held in memory for this visit, if any.

### Behaviour without consent, and its UX cost

Without consent, every choice lives in module memory. It survives HTMX swaps and enhanced navigation within the same document, because the listeners are bound once and reapply state after each swap. It does not survive a full load, a reload or a new tab.

- **Theme and palette:** the page returns to the OS scheme and the default palette.
- **Bionic reading:** it returns to off.
- **Study Board:** the board is forgotten unless its `?board=` URL is open. The shelf disappears after a reload.

The visitor accepted this cost by declining optional storage. The consent copy and the appearance panel note say so plainly.

### Copy

The notice and the preferences dialog name everything the category covers: the search toolbar, colour scheme, palette, reading aid and Study Board. They also say that DENY or switching the category off deletes these settings.

The appearance panel note stopped claiming that preferences "are kept on this device". It now says the choices apply to this visit unless preference storage is allowed in Cookie settings.

The Study Board lede now says the board is remembered only when preference storage is allowed.

The privacy policy is database content (`static_pages`, slug `privacy-policy`). It describes cookies generically and lists no individual cookie or storage key, so it needs no change.

## Risks / Trade-offs

- **The scheme flashes after a reload without consent.** There is no flash in the strict sense: the head resolver already applies the OS scheme. An explicit choice simply does not survive the reload.
- **Ordering with the unarchived `sync-design-2026-09-29` change.** That change also modifies the consent requirement. This change's delta carries the full requirement text, including its three-action rule, so archiving after it yields the intended text. The PR that merges second must archive against the updated `main`.
