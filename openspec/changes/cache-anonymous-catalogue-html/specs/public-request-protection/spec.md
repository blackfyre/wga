## MODIFIED Requirements

### Requirement: Dynamic responses remain private at the edge

Personalised responses, HTMX responses, responses rendered from cookie-derived state, error responses, and non-catalogue routes SHALL remain private at the edge. Generated `/agents/*` Markdown and `/llms.txt` SHALL be explicitly public, SHALL omit `Set-Cookie`, and MAY be cached separately from HTML. Existing safe static-asset caching MAY continue.

#### Scenario: Dynamic catalogue HTML is returned

- **WHEN** WGA returns a response to a request carrying `HX-Request`
- **THEN** the response is `private, no-store` and the edge cache rule bypasses the request

#### Scenario: Generated agent content is returned

- **WHEN** Cloudflare receives a successful `/agents/*` or `/llms.txt` response
- **THEN** the response is eligible for the dedicated public agent-content cache policy and cannot share a cache entry with HTML

## ADDED Requirements

### Requirement: Anonymous catalogue HTML is edge-cacheable

Catalogue route families that opt in SHALL send anonymous, non-HTMX `GET` and `HEAD` 200 HTML responses to requests that negotiate `text/html`, rendered without cookie-derived state, as `Cache-Control: public, max-age=0, s-maxage=300` without `Set-Cookie`, and these responses SHALL contain no session token, itinerary tray content, or other per-visitor state. The Cloudflare configuration SHALL make the opted-in path families cache-eligible while respecting origin cache headers, and SHALL bypass the cache for requests carrying an `HX-Request` header, an `Accept` header that asks for `text/markdown`, or the `wga_aw_prefs` cookie. Catalogue changes SHALL become visible at the edge within five minutes without a purge.

#### Scenario: Anonymous visitor requests a catalogue page

- **WHEN** a request without `HX-Request`, a Markdown `Accept` header, or the `wga_aw_prefs` cookie asks for an opted-in catalogue page that renders successfully
- **THEN** the response is publicly cacheable for at most 300 seconds, carries no `Set-Cookie`, and contains no per-visitor state

#### Scenario: A flood repeats a cached catalogue URL

- **WHEN** many clients request the same cached catalogue URL within its edge lifetime
- **THEN** Cloudflare answers them from cache without forwarding them to WGA

#### Scenario: Catalogue content is edited

- **WHEN** an editor changes published catalogue content
- **THEN** the change is visible to anonymous visitors within five minutes
