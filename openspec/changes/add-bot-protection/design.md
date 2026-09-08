## Context

See `proposal.md` for motivation. WGA is one PocketBase process backed by SQLite on a four-CPU, 10 GB Railway instance. During the observed crawler burst, most requests reached dynamic catalogue routes, many clients disconnected after long waits, and memory was already too constrained to justify a broad in-process response cache.

`beta.wga.hu` has been placed behind Cloudflare on the Free plan, but the zone is otherwise essentially unconfigured. During planning, `1.1.1.1` returned Cloudflare proxy addresses while `8.8.8.8` and the local resolver still returned the Railway origin, so rollout must prove DNS convergence rather than assuming that proxying is universal. The Railway hostname remains public even after DNS converges.

The application currently publishes a sitemap-only `robots.txt`, has a bounded in-memory limiter only for guestbook submissions, and resolves anonymous-write identities through `internal/requesttrust`. Public catalogue reads use app-scoped PocketBase query methods, so HTTP cancellation cannot interrupt an in-flight query; it can only prevent subsequent stages.

Cloudflare Free provides one rate-limit rule, IP-only counting, a ten-second counting and mitigation period, and a block action. Free Bot Fight Mode is a separate whole-zone toggle that cannot be skipped by custom rules and can produce false positives. Request Header Transform Rules can overwrite a client-supplied custom header before proxying to the origin.

Cloudflare's runtime Markdown for Agents conversion is available only on paid plans. WGA will not upgrade solely for that feature; it will publish its own generated Markdown so Cloudflare Free can cache already-safe agent resources without dynamically converting HTML.

## Goals / Non-Goals

**Goals:**

- Authenticate Cloudflare-originated catalogue traffic before trusting visitor identity.
- Reject direct-origin and excess catalogue work before database access.
- Use the strongest suitable Cloudflare Free controls without requiring a paid plan.
- Bound both per-client bursts and total concurrent dynamic reads.
- Preserve canonical indexing and normal progressive-enhancement behaviour.
- Give cooperative agents a low-cost, discoverable alternative to dynamic HTML.
- Make application thresholds configurable and measurable without logging raw addresses or secrets.
- Stop cancelled multi-stage workflows at the earliest safe boundary.

**Non-Goals:**

- Upgrading Cloudflare unless future evidence shows a requirement unavailable on Free is essential.
- Accurately classifying every client as human or bot inside WGA.
- Building a distributed rate limiter for multiple application replicas.
- Caching full personalised or HTMX HTML responses.
- Performing request-time HTML-to-Markdown conversion or publishing one unbounded full-catalogue document.
- Interrupting a PocketBase query that has already entered SQLite.
- Treating Host comparison or a publicly supplied forwarding header as origin authentication.

## Decisions

### Layer Cloudflare Free controls, authenticated origin ingress, and application admission

Cloudflare will provide automatic DDoS protection, one route-focused rate-limit rule, and an evaluated Bot Fight Mode rollout. WGA will independently require an authenticated Cloudflare origin marker for protected canonical catalogue traffic, validate the canonical Host as a routing invariant, and enforce bounded application admission.

An edge-only design was rejected because Cloudflare counters can lag by several seconds, rotating sources can remain below a per-IP threshold, and the Railway origin is publicly reachable. An application-only design was rejected because rejected requests would still consume origin connections and runtime resources.

### Authenticate Cloudflare with an overwritten origin header

A Cloudflare Request Header Transform Rule will set and overwrite `X-WGA-Edge-Secret` for requests to `beta.wga.hu`. WGA will compare its value in constant time with `WGA_CLOUDFLARE_EDGE_SECRET`, loaded and validated through `internal/config`. The secret will be high entropy, redacted by configuration and logging helpers, excluded from request logs and Sentry, and independently rotatable by temporarily accepting a keyring of current and next values.

For protected catalogue traffic in staging and production:

1. A non-canonical Host returns `421 Misdirected Request`.
2. A missing or invalid edge secret returns `403 Forbidden`, even if the Host is spoofed correctly.
3. Only an authenticated request proceeds to trusted visitor identity and admission.

Health, static assets, sitemap files, and `robots.txt` remain exempt so Railway can operate the deployment directly. Host checking alone was rejected as an authentication boundary because a direct client can choose its HTTP Host header.

Authenticated Origin Pulls were not selected because Railway's managed public endpoint does not expose origin TLS client-certificate verification. Cloudflare Tunnel was not selected because it would add another deployed runtime and change the Railway network topology. Both remain future alternatives if Railway gains a suitable private-origin integration.

### Add a Cloudflare-via-Railway trusted identity source

`internal/requesttrust` will add a `cloudflare-railway` source for protected reads. It will require exactly one valid Railway edge marker, valid Cloudflare origin authentication, and exactly one syntactically valid `CF-Connecting-IP`. It will ignore `X-Forwarded-For` and `X-Real-IP` in this mode. Development and tests retain the direct source.

This ordering is essential: `CF-Connecting-IP` is authoritative only on authenticated Cloudflare-to-origin traffic. Railway's edge marker alone does not prove Cloudflare involvement because direct Railway requests also traverse Railway's edge.

The protection package will convert the resolved address to a process-private keyed digest immediately. Raw addresses, proxy headers, digests, and origin secrets will not be persisted or logged.

### Centralise application protection policy in an owning package

A new `internal/requestprotection` package will own route classification, rate windows, bounded identity retention, concurrent capacity, cancellation checkpoints, and decision types. Registration middleware will remain a thin adapter: validate ingress, obtain trusted identity, invoke admission, map the result to HTTP, defer release, and log the decision.

The in-memory identity table will have a fixed maximum and deterministic expiry/oldest-entry eviction. Global capacity protects the process if distributed sources churn through the identity table. Copying the guestbook limiter into each handler was rejected because route-wide capacity and consistent privacy behaviour require one policy owner.

### Use separate application profiles and one global capacity budget

Application routes will be classified before handler execution:

- Search profile: `/artists`, `/artworks`, and `/artworks/results`.
- Fragment profile: `/dual-mode`.
- Detail profile: `/artists/{name}`, `/artists/{name}/{awid}`, and generated `/agents/artists/{id}.md` or `/agents/artworks/{id}.md` cache misses.

Separating `/dual-mode` prevents its automatic fragment traffic from consuming a legitimate visitor's search allowance. Static assets, sitemap files, `robots.txt`, health, administrative routes, and write workflows remain outside this read-admission policy. The global capacity token is acquired without an unbounded wait; failure returns `503`. Per-client exhaustion returns `429`. Both responses use minimal plain bodies with `Retry-After` and do not invoke Templ.

Initial staging values will be configurable and conservative: eight concurrent protected reads, 20 search requests per minute, 30 dual-mode fragment requests per minute, and 60 detail requests per minute per identity. Observe mode updates the same bounded counters and records would-be decisions but neither rejects nor reserves scarce concurrency after determining that enforcement would have rejected. Verification under representative traffic must precede production enforcement.

### Use the single Cloudflare Free rate rule for burst absorption

The one Free rate-limit rule will exclude Cloudflare-verified bots and match only expensive dynamic paths:

```text
not cf.bot_management.verified_bot and (
  http.request.uri.path eq "/artists" or
  starts_with(http.request.uri.path, "/artists/") or
  http.request.uri.path eq "/artworks" or
  http.request.uri.path eq "/artworks/results" or
  http.request.uri.path eq "/dual-mode" or
  starts_with(http.request.uri.path, "/agents/")
)
```

The initial threshold will be 15 requests per IP per ten seconds, followed by the Free-plan ten-second block. This is a burst absorber, not a precise origin request cap; Cloudflare documents that enforcement can lag. Security Events and Railway traffic evidence will be used to tune it without assuming paid-plan managed challenges, custom counting, NAT-aware identity, or longer periods.

Bot Fight Mode will be enabled separately only after baseline functional checks. Because Free Bot Fight Mode cannot be skipped or scoped, Security Events and browser checks must confirm it does not block legitimate monitoring, accessibility tools, or WGA interactions. Its rollback is simply disabling Bot Fight Mode; the rate rule and application protection remain active.

Browser Integrity Check and Full (strict) origin TLS are recommended zone baseline settings. Always Use HTTPS is also recommended after confirming canonical HTTPS behaviour. Under Attack Mode is reserved for a short emergency because an interstitial challenge can disrupt HTMX and normal browsing.

### Publish generated Markdown rather than converting HTML at request time

WGA will publish a concise `/llms.txt` that describes the collection, links the canonical sitemap, documents the agent-resource URL patterns, and points to usage guidance. It will not publish `llms-full.txt`: combining the full catalogue would create a large bulk-export surface and an expensive generation artefact.

A new generated-publication package will build deterministic Markdown from bounded public artist and artwork projections, not from rendered HTML. It will write only published fields, canonical URLs, attribution, and related canonical links. It will exclude itinerary state, cookies, administrative fields, raw record dumps, embedded scripts, and private persistence metadata.

The generator will follow the existing sitemap lifecycle: serialised generation, a staging directory under `app.DataDir()`, validation, atomic publication, stale-file pruning, startup/scheduled/manual execution, and structured run logging. Files will be served from durable generated storage rather than the embedded build filesystem:

- `/agents/artists/{id}.md`
- `/agents/artworks/{id}.md`

An unavailable or unpublished ID therefore has no generated file and returns 404 without a PocketBase query. Generated responses use `Content-Type: text/markdown; charset=utf-8`, an explicit public cache policy, a canonical `Link`, and no `Set-Cookie`.

Canonical artist and artwork handlers will inspect `Accept` before database lookup. A request that prefers `text/markdown` will receive a temporary redirect to the stable generated path with `Vary: Accept`. Ordinary HTML will continue unchanged and advertise the same path through `Link: <...>; rel="alternate"; type="text/markdown"` and document metadata. This covers agents that negotiate Markdown and agents that discover explicit resources while keeping one cacheable representation.

Cloudflare Free will use a dedicated Cache Rule for successful `/agents/*` and `/llms.txt` responses. It will not cache 404s, responses carrying `Set-Cookie`, or canonical HTML/HTMX. Generated cache misses remain in the detail admission profile even though serving the file does not query PocketBase.

### Keep crawler guidance and content policy ownership unambiguous

The application-owned `robots.txt` will continue advertising the canonical sitemap and add exclusions for `/dual-mode`, `/artworks/results`, and query-string variants. Canonical artist, artwork, `/llms.txt`, and generated agent resources remain crawlable. Cloudflare Managed `robots.txt` will remain disabled during this change to avoid silently prepending a second policy. AI Crawl Control and `Content-Signal` values remain separate content-rights decisions and are not inferred as part of availability protection.

Cloudflare caches static assets under its normal extension-based behaviour, but no Cache Rule will make HTML or HTMX fragments eligible. Public pages participate in itinerary session projection, and responses with `Set-Cookie` indicate personalised state.

### Emit bounded structured operational events

Request-scoped logging will emit stable event names for host rejection, origin-authentication rejection, client-rate rejection, capacity rejection, and cancellation checkpoints. Fields will include route profile, decision, HTTP status, configured limit, and bounded utilisation. They will exclude raw IPs, request slugs, proxy headers, limiter keys, and secret values. Expected admission and cancellation outcomes will not be reported to Sentry as server faults.

## Risks / Trade-offs

- [Cloudflare DNS has not converged for every resolver] → Verify authoritative and multiple public resolvers plus `CF-Ray`/`/cdn-cgi/trace` before treating the proxy as active.
- [The origin secret could be disclosed] → Use high entropy, overwrite the header at Cloudflare, redact it everywhere, support two-key rotation, and reject protected traffic on mismatch.
- [Free Bot Fight Mode can block legitimate automation and cannot be skipped] → Enable only after baseline checks, inspect Security Events, and disable it independently on false positives.
- [The single Free rate rule cannot express different search and detail thresholds] → Use a generous burst threshold at Cloudflare and retain separate application profiles.
- [Clean Markdown makes bulk collection easier] → Publish only already-public fields, omit a full-catalogue document, retain robots guidance, rate-limit cache misses, and monitor edge request volume.
- [Generated Markdown can become stale or partially published] → Reuse the sitemap's staging, validation, atomic publication, and stale-pruning pattern.
- [Legitimate users behind a shared NAT may share rate buckets] → Keep Cloudflare and application thresholds above normal interaction bursts, include `Retry-After`, and tune from aggregate evidence.
- [Distributed bots can rotate addresses and churn bounded state] → Rely on global application capacity as the final resource bound rather than treating per-client limits as sufficient.
- [In-memory state resets on deploy and is not shared across replicas] → Accept this for the current single-replica deployment; require a new design before scaling replicas.
- [Route patterns can be misclassified as handlers evolve] → Use explicit route-pattern tests and fail unclassified routes open unless deliberately assigned.

## Migration Plan

1. Confirm `beta.wga.hu` is orange-cloud proxied and DNS has converged across authoritative, Cloudflare, Google, and local resolvers.
2. Configure Full (strict), Browser Integrity Check, and the static `X-WGA-Edge-Secret` overwrite rule; install the matching secret in Railway without logging it.
3. Add validated protection/trust settings and deploy the application in observe mode with two-key origin-secret support.
4. Verify authenticated Cloudflare requests resolve `CF-Connecting-IP`, spoofed/direct-origin requests fail before database work, and health/static routes remain available.
5. Add application admission, crawler rules, cancellation checkpoints, and structured telemetry; load-test observe-mode decisions and establish thresholds.
6. Add generated Markdown publication and discovery, verify atomic rebuilds, then configure the agent-only Cloudflare cache rule and confirm origin bypass on cache hits.
7. Enable application enforcement in staging, then create the single Free rate-limit rule and verify its block/verified-bot behaviour through Cloudflare Security Events.
8. Establish a legitimate-traffic baseline, enable Bot Fight Mode, and disable it if functional checks or Security Events show unacceptable false positives.
9. Enable production enforcement after staging evidence is accepted.

Application rollback consists of setting protection mode to `observe` or `off`. Cloudflare rollback consists of disabling Bot Fight Mode and/or the rate-limit rule; the origin header rule must remain enabled while application origin authentication is enforced. During secret rotation, configure WGA to accept old and new values, update Cloudflare to the new value, verify it, then remove the old value.
