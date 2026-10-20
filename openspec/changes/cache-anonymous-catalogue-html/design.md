# Design

## Context

Cloudflare Free ignores `Vary` other than `Accept-Encoding`, and custom cache keys on headers or cookies are not available, so one URL cannot be cached separately per HTMX variant or per cookie. Cache Rules can, however, match request headers and cookies to bypass the cache, and Cloudflare does not cache a response that carries `Set-Cookie`.

## Decisions

### Per-visitor itinerary state loads after the document

The cached document renders stateless add controls (`ADD TO ITINERARY`, no `_csrf`, the existing `data-itinerary-add-form` hook) and an empty `#itinerary-tray` placeholder with `hx-get="/itineraries/session"` and `hx-trigger="load"`. The session route renders the existing templ tray; its root carries `data-itinerary-csrf`, `data-itinerary-count`, and `data-itinerary-added` (at most fifteen artwork IDs). The placeholder pins `hx-target="this"`, its own indicator, and `transition:false`, so the background load neither inherits the body's main-area swap and progress indicator nor runs a document view transition. One delegated module in `resources/js/itinerary.ts` injects `_csrf` into add requests on `htmx:configRequest` and applies the added and full state to add controls after the tray settles and after every `htmx:afterSettle`, reusing the existing client-side control-state machinery. Nothing binds to swapped nodes, so the behaviour survives HTMX swaps.

Rejected: per-control out-of-band fragments (page-specific ID lists and many swaps) and client-rendered tray markup (duplicates templ).

### The session is created on the first addition

The router-wide itinerary middleware is removed. Without a session cookie, `/itineraries/session` returns an empty tray and sets no cookie, so catalogue page views and floods no longer mint sessions. Private itinerary pages whose forms need a session (the builder, the Study Board import, and the publication confirmation) still issue one when opened, because their token-bound mutations, such as importing a board, have no first-addition path. Pages under `/itineraries` are private and render an existing session on the server, so the tray remains available there without JavaScript.

A first addition without a session cookie has no session secret to protect, so it is accepted when the request positively asserts the same origin (a matching `Origin` or `Referer`, or `Sec-Fetch-Site: same-origin`), and the session cookie (`SameSite=Lax`) is issued only after the addition succeeds. Two concurrent first additions from one browser can each create a session; the later cookie wins and the earlier draft is left for the purge job. This race is accepted. When a session cookie is present, the HMAC `_csrf` token remains required. A no-JavaScript POST that has a cookie but no token receives an uncached confirmation page containing a token-bearing form. The HTMX form of the same case, typically an addition made before the session tray loaded, receives the session tray and a retry toast.

### Shared caching is explicit opt-in

A router middleware marks a response `Cache-Control: public, max-age=0, s-maxage=300` only for anonymous, non-HTMX `GET`/`HEAD` 200 responses to requests that negotiated `text/html`, rendered without cookie-derived state, and route families opt in explicitly through one list; the decision is made when the status is written, so cookies and explicit policies set by the handler take precedence. HTMX responses and responses where remembered artwork-search preferences were applied send `private, no-store`. Errors, 4xx/5xx, previews, `/itineraries/*`, postcards, guestbook, feedback, `/_/`, and `/api/` remain private.

### Edge rules

One Cache Rule marks the opted-in path families as eligible and respects origin cache headers, excluding requests with an `HX-Request` header, an `Accept` header asking for `text/markdown` (generated-Markdown negotiation shares the document URLs), or a `wga_aw_prefs` cookie. PocketBase authentication does not use cookies on these routes. Browser TTL respects the origin. Invalidation is TTL-only; purge-everything is the manual escape hatch. Hook-driven purge is deferred because it adds an API credential and a retried third-party side effect to content writes.

## Risks

- A public response that accidentally contains session or consent-dependent markup would be shared across visitors. Go tests assert that opted-in responses, including `HEAD`, carry no `Set-Cookie`, CSRF value, or tray content.
- Tray and control state appear one request after first paint; the tray reserves space only when the count is above zero.
- The cookie-consent banner must remain client-driven; this is checked during implementation.
- The first-addition CSRF rule relies on `SameSite=Lax` and the Origin check; it requires a security review.
- `/itineraries/session` becomes an uncached path that floods can target; it is protected by origin admission and the edge rate-limit rule.
- Unpublished or edited catalogue content can remain visible at the edge for up to five minutes.

Cloudflare documentation confirms that Cache Rule expressions can match `http.cookie` and `http.request.headers`, and that with Origin Cache Control (always enabled below Enterprise) a response carrying `Set-Cookie` is not cached.
