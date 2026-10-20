# Proposal

## Why

Distributed scrapers rotate residential addresses and send one or two requests each, so per-client limits never engage and every request reaches WGA. Admission now sheds that load, but each shed request still reaches the origin, and the scraper can move to any other catalogue route. Anonymous catalogue HTML is identical for every visitor apart from the itinerary session layer, so letting Cloudflare cache it would absorb such floods at the edge regardless of which route is targeted.

Today every public HTML response carries per-visitor state: the itinerary middleware issues a session cookie on every page view, renders the session's CSRF token, tray, and added-artwork state into the page, and marks the response `private, no-store`. The artwork search also renders remembered presentation choices from the `wga_aw_prefs` cookie.

## What Changes

- Catalogue HTML documents carry no per-visitor state: add controls render without a CSRF token or session-specific state, and the layout renders an empty tray placeholder.
- The tray, CSRF token, and added/full state load after the document from an uncached `GET /itineraries/session` fragment and are applied to add controls by one delegated, swap-safe script.
- The itinerary session cookie is issued on the first addition, not on page views. A first addition without a session cookie is accepted on a same-origin request without a CSRF token; requests with a session cookie keep the existing token requirement. A no-JavaScript addition that lacks a token receives an uncached confirmation form instead of a 403.
- Anonymous, non-HTMX, cookie-independent `GET`/`HEAD` 200 catalogue HTML is sent as `public, max-age=0, s-maxage=300` without `Set-Cookie`, opted in per route family. HTMX responses and responses rendered from `wga_aw_prefs` remain `private, no-store`.
- One Cloudflare Cache Rule makes the opted-in path families cache-eligible, respecting origin cache headers, and bypasses requests that carry `HX-Request`, ask for Markdown, or carry the `wga_aw_prefs` cookie. Catalogue edits become visible at the edge within five minutes; purge-everything is the manual escape hatch.
- `GET /itineraries/session` joins the origin admission profiles and the Cloudflare rate-limit route families.

## Capabilities

### Modified Capabilities

- `public-request-protection`: anonymous catalogue HTML becomes edge-cacheable under explicit eligibility and bypass rules, replacing the requirement that all dynamic HTML remains private at the edge.
- `visitor-itineraries`: the session is created on first addition, and tray and control state load after the document.
- `catalogue-exploration`: responses rendered from remembered presentation choices are never shared-cached.

## Impact

`internal/handlers/itinerary_session.go`, `internal/handlers/itineraries/`, `internal/itineraries/` (first-addition CSRF rule), `internal/assets/templ/layouts/layout.templ`, `internal/assets/templ/components/add_to_itinerary.templ` and its call sites, `internal/handlers/artworks/`, a new cache-policy helper in `internal/handlers`, `resources/js/itinerary.ts`, `internal/requestprotection/classification.go`, Playwright and Go tests, and the Cloudflare zone configuration in `docs/bot-protection-runbook.md`. The CSRF change requires a security review before release.
