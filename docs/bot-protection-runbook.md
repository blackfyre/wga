# Bot Protection Runbook

## Purpose

Use this runbook to operate WGA's Cloudflare Free and Railway request-protection boundary. Apply changes to staging first. Never record origin secrets, raw client addresses, proxy headers, cookies, query strings, or catalogue slugs in tickets, logs, screenshots, or command output.

The trusted request path is:

```text
visitor -> Cloudflare -> Railway edge -> WGA
```

Cloudflare must overwrite `X-WGA-Edge-Secret`. WGA authenticates that value before trusting the single `CF-Connecting-IP` supplied by Cloudflare. Railway's edge marker is also required. Application protection mode does not disable this origin-authentication check.

## Required configuration

### Railway variables

Configure these through Railway's secret and variable controls:

```text
WGA_CLIENT_IP_SOURCE=cloudflare-railway
WGA_CLOUDFLARE_EDGE_SECRET=<current unpadded Base64URL 32-byte secret>
WGA_CLOUDFLARE_EDGE_SECRET_NEXT=<optional rotation overlap secret>
WGA_PUBLIC_REQUEST_PROTECTION_MODE=off|observe|enforce
WGA_PUBLIC_READ_MAX_CONCURRENT=8
WGA_PUBLIC_READ_SEARCH_PER_MINUTE=20
WGA_PUBLIC_READ_FRAGMENT_PER_MINUTE=30
WGA_PUBLIC_READ_DETAIL_PER_MINUTE=60
WGA_PUBLIC_READ_LIMITER_CAPACITY=4096
WGA_PUBLIC_READ_RETRY_AFTER=5s
```

Generate each origin secret independently with a cryptographically secure generator. It must decode as exactly 32 bytes of unpadded Base64URL. Inject it directly into Railway and Cloudflare; do not commit it or print it during verification.

Railway Under Attack Mode must normally remain disabled while Cloudflare owns the public bot challenge. Check it with:

```sh
railway waf under-attack status --service wga --environment uat --json
```

An active Railway challenge returns pre-origin `429` responses to non-browser monitoring and prevents WGA or Cloudflare-origin-authentication evidence from being observed.

### Cloudflare zone

For `beta.wga.hu` in staging:

- keep the CNAME orange-cloud proxied with TTL Auto;
- use SSL/TLS Full (strict);
- enable Always Use HTTPS only after canonical HTTPS succeeds;
- enable Browser Integrity Check;
- keep Cloudflare Managed `robots.txt` disabled;
- make the Request Header Transform Rule match `http.host eq "beta.wga.hu"` and **set**, rather than add, `X-WGA-Edge-Secret` to the current secret;
- keep one Free rate-limit rule with action `block`, 15 requests per IP per ten seconds, and a ten-second mitigation timeout;
- keep Bot Fight Mode independently switchable;
- keep one cache rule limited to `/agents/*` and `/llms.txt`, with cache and browser TTLs respecting origin headers.

The rate-limit expression is:

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

The cache-rule expression is:

```text
starts_with(http.request.uri.path, "/agents/") or
http.request.uri.path eq "/llms.txt"
```

Do not add response status TTL overrides. Successful generated resources publish `public` cache headers, while misses publish `private, no-store`. Cloudflare must respect those origin decisions.

## Generated-publication recovery

Sitemap and agent Markdown generation share one serial, atomic publication lifecycle under the configured WGA data directory. A failed generation before the current-version marker changes leaves the last complete publication selected. Do not edit generated Markdown, manifests, version directories, or the current marker manually.

For a manual rebuild, run the built command against the intended deployment data directory:

```sh
dist/wga --dir <data-directory> generate-sitemap
```

Verify the structured completion event reports bounded sitemap and agent-resource counts, the canonical sitemap and representative generated resources are available, and stale resources have been pruned. If generation fails, retain the selected publication, diagnose the redacted error type and controlled stage, correct the source or storage fault, and rerun the complete command. Restore the data directory from a verified backup only when no complete publication remains; never repair a partial version by changing the marker.

## DNS and proxy gate

Do not enable application enforcement until every path below returns only Cloudflare anycast A and AAAA addresses:

1. Discover the current authoritative nameservers with `dig +short NS wga.hu` and query each one.
2. Query `beta.wga.hu` through `1.1.1.1`.
3. Query it through `8.8.8.8`.
4. Query it through the operator's normal local resolver.
5. Check both A and AAAA answers.

```sh
dig +short A beta.wga.hu @1.1.1.1
dig +short AAAA beta.wga.hu @1.1.1.1
dig +short A beta.wga.hu @8.8.8.8
dig +short AAAA beta.wga.hu @8.8.8.8
dig +short A beta.wga.hu
dig +short AAAA beta.wga.hu
curl --fail --silent --show-error --dump-header - --output /dev/null https://beta.wga.hu/health
curl --fail --silent --show-error https://beta.wga.hu/cdn-cgi/trace
```

Stop if any answer exposes a Railway CNAME or address, or if HTTPS lacks `server: cloudflare`, `CF-Ray`, and a successful `/cdn-cgi/trace`. Flush or replace a stale recursive resolver and repeat every check; changing or toggling an already-correct Cloudflare record does not repair an upstream resolver cache.

## Staged rollout gates

Advance only when the current gate has deterministic evidence:

1. **Proxy:** DNS and HTTPS pass the proxy gate above.
2. **Origin authentication:** deploy `cloudflare-railway` identity settings and the matching Cloudflare transform. A canonical protected request succeeds, a visitor-supplied secret is overwritten, and a direct-origin request with an invalid secret returns `403` before WGA's handler.
3. **Observe:** set `WGA_PUBLIC_REQUEST_PROTECTION_MODE=observe`. Exercise normal browser, keyboard, HTMX, monitoring, crawler, and generated-content flows. Review aggregate would-reject events and tune only from evidence.
4. **Cloudflare rate limit:** send a controlled staging burst. The first 15 requests in ten seconds may reach Railway; excess requests must return Cloudflare `429` with `Retry-After: 10`, appear as `ratelimit` blocks in Security Events, and be absent from Railway origin logs. Confirm the rule expression still excludes verified bots.
5. **Generated cache:** request `/llms.txt` and one published `/agents/*` resource twice with a unique harmless cache key. Each must transition from `MISS` to `HIT`. Repeated generated-resource 404s must remain `BYPASS` or otherwise never become `HIT`; full-page HTML, HTMX, and `Set-Cookie` responses must remain `DYNAMIC` or `BYPASS`.
6. **Bot Fight Mode:** record the functional baseline before enabling it. Repeat monitoring, accessibility, browser, HTMX, robots, sitemap, and canonical crawler checks after enabling it. Review Security Events for false positives and prove that Bot Fight Mode can be disabled without changing the rate-limit or application controls.
7. **Enforce:** after the observation evidence is accepted, set `WGA_PUBLIC_REQUEST_PROTECTION_MODE=enforce` in staging. Verify direct-origin, host, rate, and capacity rejection responses, then repeat the legitimate-traffic checks before production rollout.

Record the deployment ID and release commit for each Railway gate. A successful command exit is not deployment evidence; the named deployment must reach `SUCCESS`.

## Threshold tuning

Use aggregate observe-mode decisions, Cloudflare Security Events, Railway request counts, memory, CPU, and latency over the agreed observation interval. Keep Cloudflare's single burst threshold above ordinary page and HTMX interaction bursts, then tune WGA's search, fragment, and detail profiles independently. Change one limit at a time and repeat the same representative traffic before accepting it.

Increase per-client limits only when legitimate shared-network traffic is rejected and capacity evidence remains safe. Lower concurrency or profile limits when database latency, memory, or cancellation evidence shows resource pressure. Keep limiter capacity bounded; a larger entry budget trades memory for resistance to rotating client identities. Record the old value, new value, aggregate reason, deployment ID, and verification outcome without recording raw identities or request values.

## Origin-secret rotation

Rotate without an authentication gap:

1. Generate a new independent 32-byte unpadded Base64URL secret without printing or logging it.
2. Keep the old secret in `WGA_CLOUDFLARE_EDGE_SECRET`, set the new secret in `WGA_CLOUDFLARE_EDGE_SECRET_NEXT`, and deploy WGA.
3. Verify both values are accepted at the Railway origin while invalid values remain rejected.
4. Change the Cloudflare transform rule to overwrite the header with the new secret.
5. Verify canonical Cloudflare requests succeed and visitor-supplied values are still overwritten.
6. Promote the new value to `WGA_CLOUDFLARE_EDGE_SECRET`, clear `WGA_CLOUDFLARE_EDGE_SECRET_NEXT`, and deploy again.
7. Verify canonical traffic and invalid-secret rejection, then remove every temporary local copy of the old and new values.

If a rotation fails, restore the transform to the old value while WGA still accepts both. Do not remove the old value until Cloudflare's new value has been verified through the canonical path.

## Rollback order

Rollback the layer shown by the evidence, preserving independent controls:

1. Disable Bot Fight Mode first for browser or automation false positives.
2. Disable the Cloudflare rate-limit rule for incorrect edge `429` responses.
3. Disable the agent cache rule for stale or incorrectly eligible generated responses; purge only affected agent URLs when necessary.
4. Change application mode from `enforce` to `observe`, then to `off` only if application admission is responsible.
5. For an origin-secret mismatch, restore the previously verified transform value during the two-key overlap. If no accepted key remains, restore `WGA_CLIENT_IP_SOURCE=railway` and deploy before removing the Cloudflare-only variables.

Keep the origin-header transform enabled while WGA uses `cloudflare-railway`, even when protection mode is `observe` or `off`. Do not grey-cloud the DNS record as a routine rollback; that restores a direct origin path and removes Cloudflare controls.

## Privacy-safe evidence

Retain only what is needed to approve or reject a rollout:

- deployment ID, release commit, status, and observation interval;
- Cloudflare rule ID, bounded route class, action, status, and aggregate event count;
- WGA event name, protection mode, bounded profile, decision, configured limit, and bounded capacity use;
- aggregate origin request counts proving that edge-blocked requests did not arrive;
- cache status sequence and response cache policy for representative generated, missing, HTML, HTMX, and `Set-Cookie` responses.

Redact or omit raw IP addresses, secret values and hashes, request and proxy headers, cookies, query strings, catalogue slugs, limiter keys, personal data, and response bodies. Expected protection rejections and cancellations are operational outcomes, not Sentry server faults.
