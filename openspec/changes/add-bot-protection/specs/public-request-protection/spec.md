## Purpose

Protects WGA's public catalogue from automated overload while preserving canonical discovery, legitimate browsing, and privacy-safe operations.

## ADDED Requirements

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

The system SHALL apply separate admission profiles to interactive catalogue search routes and canonical catalogue detail routes. Admission SHALL combine a bounded per-client request rate with a global concurrent-work limit and SHALL not create an unbounded queue or identity store.

#### Scenario: Client exceeds its route-profile rate

- **WHEN** a trusted client identity exceeds the configured request allowance for a protected route profile
- **THEN** the system rejects the request before catalogue database work with HTTP 429 and a `Retry-After` header

#### Scenario: Catalogue capacity is exhausted

- **WHEN** the configured concurrent-work capacity for protected catalogue requests is exhausted
- **THEN** the system rejects new protected work immediately with HTTP 503 and a `Retry-After` header

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

The system SHALL emit structured telemetry for rejected admission and abandoned protected work, including the route profile, decision, status, and current capacity without including a raw client address or requested record slug.

#### Scenario: Request is rate limited

- **WHEN** admission rejects a protected request for exceeding its client allowance
- **THEN** one structured event records the rate-limit decision and route profile without raw client data

#### Scenario: Request is load shed

- **WHEN** admission rejects a protected request because concurrent capacity is exhausted
- **THEN** one structured event records the capacity decision and bounded utilisation values

#### Scenario: Request cancellation is observed

- **WHEN** a protected workflow stops at a cancellation checkpoint
- **THEN** one structured event records the cancellation stage and route profile without reporting it as a server fault

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

The Cloudflare configuration SHALL NOT make personalised or HTMX HTML responses cache-eligible as part of this change. Generated `/agents/*` Markdown and `/llms.txt` SHALL be explicitly public, SHALL omit `Set-Cookie`, and MAY be cached separately from HTML. Existing safe static-asset caching MAY continue.

#### Scenario: Dynamic catalogue HTML is returned

- **WHEN** Cloudflare receives a full-page or HTMX catalogue response
- **THEN** the response is not stored by a new bot-protection cache rule

#### Scenario: Generated agent content is returned

- **WHEN** Cloudflare receives a successful `/agents/*` or `/llms.txt` response
- **THEN** the response is eligible for the dedicated public agent-content cache policy and cannot share a cache entry with HTML
