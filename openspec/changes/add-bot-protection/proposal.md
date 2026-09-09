## Why

Automated traffic is exhausting the single Railway instance before legitimate requests can complete: observed load saturated four CPUs, consumed about 9.5 GB of a 10 GB memory limit, and produced sustained client cancellations with 10–30 second responses. WGA needs layered admission and load-shedding controls that preserve useful indexing while preventing crawlers from overwhelming PocketBase.

## What Changes

- Define crawler guidance that permits canonical public records while excluding interactive and query-driven endpoints from cooperative crawling.
- Admit database-backed public requests through bounded, route-aware concurrency control and reject excess work quickly with retry guidance.
- Stop multi-stage catalogue workflows promptly when their request has been cancelled.
- Authenticate Cloudflare-originated catalogue traffic before trusting `CF-Connecting-IP`, and avoid retaining raw client addresses.
- Reject direct Railway catalogue traffic even when a caller spoofs the canonical host, while keeping operational endpoints available.
- Emit structured, privacy-preserving admission and cancellation telemetry for operational tuning.
- Configure the existing Cloudflare Free proxy with its available bot and rate-limit controls while retaining application admission as defence in depth.
- Publish lightweight agent discovery and pre-generated, session-independent Markdown representations that Cloudflare Free can cache without converting or caching personalised HTML.
- Exclude HTML and HTMX response caching until personalised boundaries and memory costs are measured; cache only explicitly public agent resources.

## Capabilities

### New Capabilities

- `public-request-protection`: Trusted-origin enforcement, crawler and agent guidance, cacheable public Markdown representations, bounded public-read admission, cancellation handling, and protection telemetry.

### Modified Capabilities

None.

## Impact

- Affects server configuration, trusted request identity, route middleware, `robots.txt`, public catalogue workflows, structured logging, and associated Go and browser-independent HTTP tests.
- Adds deployment configuration for protection thresholds, Cloudflare-origin authentication, and trusted proxy-chain selection through `internal/config` and `.env.example`.
- Requires a Cloudflare Request Header Transform Rule, the single rate-limit rule available on the Free plan, Bot Fight Mode evaluation, and DNS-proxy verification. Infrastructure mutation still requires explicit credentials and operator authorisation.
- Extends the existing generated-publication lifecycle with `/llms.txt` and bounded Markdown records under `/agents/`; no Cloudflare plan upgrade is required.
- Public clients may receive `403 Forbidden`, `421 Misdirected Request`, `429 Too Many Requests`, or `503 Service Unavailable` when origin authentication, routing, or admission policy rejects them.
