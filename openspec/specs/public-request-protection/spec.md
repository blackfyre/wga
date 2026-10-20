# Public Request Protection Specification

## Purpose

Protects WGA's public catalogue from automated overload while preserving canonical discovery, legitimate browsing, and privacy-safe operations.

## Requirements

### Requirement: Cooperative crawler guidance

The system SHALL publish crawler guidance that advertises the canonical sitemap, permits canonical public content, and excludes interactive, fragment-only, and query-generated catalogue surfaces from crawling.

#### Scenario: Crawler retrieves guidance

- **WHEN** a client requests `/robots.txt`
- **THEN** the response identifies the canonical sitemap and disallows `/dual-mode`, `/artworks/results`, and query-string variants

#### Scenario: Canonical records remain discoverable

- **WHEN** a cooperative crawler evaluates a canonical artist or artwork record URL without a query string
- **THEN** the crawler guidance does not disallow that URL

### Requirement: Agent-oriented public representations

The system SHALL publish a concise `/llms.txt` discovery document and pre-generated, session-independent Markdown representations for published canonical artist and artwork records. These resources SHALL derive only from public projections, SHALL NOT contain personalised itinerary state, and SHALL NOT require Cloudflare's paid Markdown conversion feature.

#### Scenario: Agent discovers supported resources

- **WHEN** an agent requests `/llms.txt`
- **THEN** the response explains the canonical site, links the sitemap and agent resource conventions, and does not embed the complete catalogue

#### Scenario: Agent requests a published Markdown record

- **WHEN** an agent requests `/agents/artists/{id}.md` or `/agents/artworks/{id}.md` for a published record
- **THEN** the system returns deterministic `text/markdown` containing the canonical URL, public record projection, attribution, and related canonical links without a session cookie

#### Scenario: Agent requests an unavailable Markdown record

- **WHEN** an agent requests a generated path for a missing or unpublished record
- **THEN** the system returns 404 without generating the representation from PocketBase during the request

#### Scenario: Agent negotiates Markdown on a canonical record

- **WHEN** a client requests a canonical artist or artwork URL with `Accept: text/markdown`
- **THEN** the system redirects to the corresponding generated Markdown resource before building the dynamic HTML view and varies that decision by `Accept`

#### Scenario: Human requests canonical HTML

- **WHEN** a client requests a canonical artist or artwork URL without preferring Markdown
- **THEN** the HTML response remains canonical and advertises the generated Markdown resource as an alternate representation

#### Scenario: Generated catalogue changes

- **WHEN** the generated publication workflow completes after published records change
- **THEN** it atomically publishes current Markdown resources and removes stale generated resources without exposing a partial set

### Requirement: Authenticated canonical edge ingress

In staging and production, the system SHALL require protected public catalogue requests to match the configured public hostname and carry valid origin authentication applied by Cloudflare. Host matching alone SHALL NOT authenticate an edge request. Operational health checks, crawler guidance, sitemap files, and static assets SHALL remain available through the deployment hostname.

#### Scenario: Direct origin catalogue request

- **WHEN** a client requests a protected catalogue route through a non-canonical deployment hostname
- **THEN** the system rejects the request before catalogue database work with HTTP 421

#### Scenario: Direct origin request spoofs the canonical host

- **WHEN** a client reaches the Railway origin with the canonical Host but without valid Cloudflare origin authentication
- **THEN** the system rejects the request before catalogue database work with HTTP 403

#### Scenario: Authenticated canonical catalogue request

- **WHEN** a protected request has the configured public hostname and valid Cloudflare origin authentication
- **THEN** edge-ingress enforcement allows the request to continue to normal admission checks

#### Scenario: Operational route through deployment hostname

- **WHEN** Railway requests `/health` through the deployment hostname
- **THEN** host enforcement allows the health check to proceed

### Requirement: Bounded public-read admission

The system SHALL apply separate admission profiles to interactive catalogue search routes and canonical catalogue detail routes. Admission SHALL combine a bounded per-client request rate with a concurrent-work limit held separately by each protected route profile, so that exhausting one profile does not reject work on another, and SHALL not create an unbounded queue or identity store.

#### Scenario: Client exceeds its route-profile rate

- **WHEN** a trusted client identity exceeds the configured request allowance for a protected route profile
- **THEN** the system rejects the request before catalogue database work with HTTP 429 and a `Retry-After` header

#### Scenario: Catalogue capacity is exhausted

- **WHEN** the configured concurrent-work capacity of a protected route profile is exhausted
- **THEN** the system rejects new work for that profile immediately with HTTP 503 and a `Retry-After` header

#### Scenario: Another profile's capacity is exhausted

- **WHEN** one protected route profile has exhausted its concurrent-work capacity and a request for a different protected profile arrives within its client allowance
- **THEN** the request is admitted if its own profile has capacity available

#### Scenario: Capacity is available

- **WHEN** a protected request is within its client allowance and concurrent capacity is available
- **THEN** the request proceeds and releases its capacity exactly once when processing finishes

#### Scenario: Agent resource misses the edge cache

- **WHEN** an authenticated request for `/agents/artists/{id}.md` or `/agents/artworks/{id}.md` reaches WGA
- **THEN** the detail admission profile protects the origin request without performing a PocketBase record query

#### Scenario: Admission identity cannot be trusted

- **WHEN** the configured trusted client-identity contract cannot resolve a valid identity
- **THEN** the protected request fails closed without trusting public forwarding headers

### Requirement: Privacy-preserving client accounting

The system SHALL derive admission identities through the configured trusted proxy chain and SHALL retain only a one-way private identifier for rate accounting and telemetry. In the Cloudflare-via-Railway mode, the system SHALL accept `CF-Connecting-IP` only after validating Railway edge markers and Cloudflare origin authentication. Raw client addresses and origin secrets SHALL NOT be persisted or written to logs.

#### Scenario: Cloudflare-via-Railway identity is valid

- **WHEN** a protected request carries valid Railway edge markers, valid Cloudflare origin authentication, and exactly one syntactically valid `CF-Connecting-IP`
- **THEN** admission uses a private derived identity rather than the raw address

#### Scenario: Cloudflare identity lacks origin authentication

- **WHEN** a request supplies `CF-Connecting-IP` without valid Cloudflare origin authentication
- **THEN** the system rejects the identity and does not use that header for admission

#### Scenario: Public forwarding header is spoofed

- **WHEN** a client supplies `X-Forwarded-For`, `X-Real-IP`, or multiple `CF-Connecting-IP` values
- **THEN** those values do not determine the Cloudflare-via-Railway admission identity

### Requirement: Cancelled work stops between stages

Protected catalogue workflows SHALL check the request context before starting each material database, projection, related-content, or rendering stage. A cancelled or expired request SHALL stop without beginning further stages and SHALL preserve the cancellation cause for existing observability suppression.

#### Scenario: Request is cancelled during a multi-stage detail workflow

- **WHEN** the request context is cancelled after one stage completes
- **THEN** no subsequent material stage starts and the workflow returns the cancellation cause

#### Scenario: Request remains active

- **WHEN** the request context remains active through all stages
- **THEN** cancellation checks do not alter the normal response

### Requirement: Protection decisions are observable

The system SHALL emit structured telemetry for rejected admission, observe-mode admission decisions, and abandoned protected work, including the route profile, decision or stage, status, and current capacity without including a raw client address or requested record slug. Persisted events SHALL be bounded: for each event type and route profile, the system SHALL emit at most one event per sampling interval and SHALL report the number of events suppressed since the previous emission. Admission metrics SHALL continue to count every decision.

#### Scenario: Request is rate limited

- **WHEN** admission rejects a protected request for exceeding its client allowance
- **THEN** a structured event records the rate-limit decision and route profile without raw client data, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Request is load shed

- **WHEN** admission rejects a protected request because concurrent capacity is exhausted
- **THEN** a structured event records the capacity decision and bounded utilisation values, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Request cancellation is observed

- **WHEN** a protected workflow stops at a cancellation checkpoint
- **THEN** a structured event records the cancellation stage and route profile without reporting it as a server fault, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Events are suppressed during a flood

- **WHEN** further events of the same type and profile occur within the sampling interval
- **THEN** they are not persisted individually, the next emitted event of that type and profile reports how many were suppressed, and admission metrics count every decision

### Requirement: Edge defence in depth

The deployed canonical hostname SHALL remain proxied through Cloudflare. Within Free-plan capabilities, the zone SHALL use one IP-counted rate-limit rule that covers expensive search, fragment, artist-detail, artwork-detail, and generated agent-record route families while excluding Cloudflare-verified bots, and SHALL evaluate Bot Fight Mode separately. Application admission SHALL continue to protect the origin because Cloudflare rate counters can lag and distributed clients can remain below an individual edge threshold.

#### Scenario: Automated burst reaches the canonical edge

- **WHEN** a non-verified source exceeds the configured Free-plan request allowance within the supported ten-second period
- **THEN** Cloudflare blocks subsequent matching requests for the supported mitigation period before forwarding them to WGA

#### Scenario: Verified crawler reaches the canonical edge

- **WHEN** Cloudflare identifies a request as verified-bot traffic
- **THEN** the catalogue rate-limit rule excludes that request from mitigation

#### Scenario: Bot Fight Mode causes unacceptable traffic loss

- **WHEN** Security Events or functional checks show Bot Fight Mode blocking legitimate WGA traffic
- **THEN** operators can disable Bot Fight Mode without disabling the application admission policy

#### Scenario: Edge protection is unavailable

- **WHEN** a request reaches WGA without having been limited by the edge
- **THEN** authenticated-ingress and application admission controls still enforce their configured protection policy

### Requirement: Dynamic responses remain private at the edge

Personalised responses, HTMX responses, responses rendered from cookie-derived state, error responses, and non-catalogue routes SHALL remain private at the edge. Generated `/agents/*` Markdown and `/llms.txt` SHALL be explicitly public, SHALL omit `Set-Cookie`, and MAY be cached separately from HTML. Existing safe static-asset caching MAY continue.

#### Scenario: Dynamic catalogue HTML is returned

- **WHEN** WGA returns a response to a request carrying `HX-Request`
- **THEN** the response is `private, no-store` and the edge cache rule bypasses the request

#### Scenario: Generated agent content is returned

- **WHEN** Cloudflare receives a successful `/agents/*` or `/llms.txt` response
- **THEN** the response is eligible for the dedicated public agent-content cache policy and cannot share a cache entry with HTML

### Requirement: Public request volume does not amplify persistent log writes

The system SHALL NOT persist a request activity log for a request that completes without an error and with a status below 500, or that admission rejects as an expected outcome, and SHALL retain persisted logs for at most 7 days. Requests that fail with an error or end in any other 5xx response SHALL continue to be persisted.

#### Scenario: Public request succeeds

- **WHEN** a public request completes without error
- **THEN** no request activity log record is persisted

#### Scenario: Admission rejects a public request

- **WHEN** admission rejects a protected request with a 403, 421, 429, or 503 response
- **THEN** no request activity log record is persisted for it

#### Scenario: Request fails with an error

- **WHEN** a request handler returns an error
- **THEN** the request activity log record is persisted

#### Scenario: Handler renders a server fault

- **WHEN** a request handler renders a 5xx response without returning an error
- **THEN** the request activity log record is persisted

#### Scenario: Logs age beyond retention

- **WHEN** a persisted log record is older than 7 days
- **THEN** it is eligible for PocketBase's scheduled log cleanup

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
