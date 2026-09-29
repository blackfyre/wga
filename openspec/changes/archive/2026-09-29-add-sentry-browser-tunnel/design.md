# Design

## Context

The browser SDK (`@sentry/browser` 10.x) posts envelopes to `*.ingest.sentry.io`, which EasyPrivacy blocks by default (`||sentry.io^$3p`). The SDK's `tunnel` option sends the same envelopes to a first-party URL instead, and adds the DSN to each envelope header. The server must then forward them without becoming an open relay. A pre-implementation security review shaped the decisions below.

## Goals / Non-Goals

**Goals:**
- Browser events reach the configured browser project for visitors with default content blockers.
- The relay cannot be used to reach any other destination, including the server Sentry project.

**Non-Goals:**
- Changing what the browser SDK collects. Release-health session envelopes, sent once per page view and History navigation, keep flowing and are sized for below.
- Operator-tunable relay limits.
- Cloudflare configuration.

## Decisions

### Split the forwarding workflow from the HTTP adapter

`internal/observability/browser_tunnel.go` owns Sentry-specific work:
- parsing the configured browser and server DSNs;
- building the upstream URL once with `(*sentry.Dsn).GetAPIURL()`;
- validating an envelope's first header line;
- forwarding it.

A new `internal/handlers/diagnostics` package is the thin HTTP adapter. It checks size, encoding and origin, then admits the request (trusted identity, per-client rate, concurrency), calls the tunnel, and maps the result. `cmd/wga/main.go` builds the tunnel next to `observability.Configure`, and `handlers.RegisterHandlers` registers the adapter only when the tunnel exists.

- **Alternative: register the route inside observability, as `/sentry-test` does.** Rejected. That route is a non-production test hook, while this is a production public ingress with admission, which belongs with the handler modules.

### Validate the envelope DSN field by field; fix the destination at construction

The relay reads a body bounded by `http.MaxBytesReader` (256 KiB, enough for browser error envelopes, since no replay is sent) under a 5 s read deadline, before taking a capacity slot, so slow uploads cannot hold forwarding capacity. It reads the first line's `dsn` member strictly with a token decoder, rejecting case-variant or duplicate keys: `encoding/json` struct decoding folds key case and keeps the last duplicate, while Sentry reads the exact key. It parses the value with `sentry.NewDsn` and requires scheme, host, port, path, public key and project ID to equal the configured browser DSN. It separately rejects the server DSN.

The forwarded request goes to the precomputed URL with only `Content-Type: application/x-sentry-envelope`. It uses a dedicated, uninstrumented `http.Client` with a 5 s timeout and no redirects. The context is detached from client cancellation, so envelopes sent during page unload still complete. The response carries only Sentry's status, `X-Sentry-Rate-Limits`, `Retry-After`, and `Cache-Control: no-store`.

### Reuse identity and capacity primitives; keep a local rate limiter

- **Identity:** admission requires the `requesttrust.Resolver` identity. In the `cloudflare-railway` mode this also enforces Cloudflare origin authentication, so direct-origin calls fail closed.
- **Concurrency:** `requestprotection.CapacityGate` (16 concurrent forwards) bounds concurrent work without queuing.
- **Rate:** per-client rate uses a small fixed-window limiter in the diagnostics package. It allows 60 envelopes per minute per client, groups IPv6 clients by /64 so address rotation does not multiply the budget, has bounded entries, and keys clients by an HMAC with a per-process random key, as `requestprotection.RateLimiter` does.
- **Origin:** checked against the configured public URL, not the request `Host`, following the itinerary `CanonicalOrigin` precedent. When both `Origin` and `Referer` are absent the request is allowed, because browsers send `Origin` on `fetch` POSTs, and identity, origin authentication and the DSN match remain the controls.

- **Alternative: reuse `requestprotection.Policy` and `RateLimiter`.** Rejected. They are bound to the GET-only read profiles and their configuration, and reusing them would change `public-request-protection` for an unrelated capability.

### Upstream failures are expected responses

Every relay-originated 5xx (502 on transport failure, 504 on timeout, and a forwarded Sentry 5xx) is marked with `requestfailure.MarkExpectedResponse`. Otherwise the server monitor would report Sentry outages back to Sentry. Logs use `logging.RequestLogger` with an outcome enum and the upstream status only. Transport errors are redacted and never include the URL.

### Browser path is a constant

`initialiseSentry` sets `tunnel: "/diagnostics/browser"`. The route and the client share one switch, the browser DSN, so page configuration stays unchanged.

## Risks / Trade-offs

- [Session envelopes make relay traffic roughly equal to page views] → 60 per minute per client and 16 concurrent forwards. Sentry's own rate-limit headers are passed back so the SDK backs off.
- [A Cloudflare WAF or bot rule challenges POSTs to the path] → Not visible from the repository. Verify delivery on staging after deploy.
- [A deployment using `railway` rather than `cloudflare-railway` client-IP mode has no origin authentication on this route] → The DSN match and limits still bound abuse to our own browser project's quota.
- [A filter list later adds the first-party path] → The path avoids tracking keywords, and it matched none of EasyList, EasyPrivacy, the uBlock, AdGuard or Brave lists when checked.
