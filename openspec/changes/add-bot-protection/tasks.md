## 1. Protection and trust configuration

- [x] 1.1 Add `internal/config` settings for protection mode, concurrent-read capacity, search/fragment/detail per-minute limits, limiter entry capacity, and retry interval; document them in `.env.example` and verify focused default/parsing/validation tests pass.
- [x] 1.2 Add a redacted current/next Cloudflare origin-secret keyring and `cloudflare-railway` client-IP source to `internal/config`; verify missing production secrets, malformed keyrings, unsupported sources, redaction, and two-key rotation tests pass.

## 2. Authenticated proxy identity

- [x] 2.1 Extend `internal/requesttrust` with a Cloudflare-via-Railway resolver that requires valid Railway edge markers and constant-time origin-secret authentication before accepting exactly one valid `CF-Connecting-IP`; verify focused valid IPv4/IPv6 and current/next secret tests pass.
- [x] 2.2 Add adversarial proxy tests proving missing/invalid secrets, duplicate or malformed `CF-Connecting-IP`, direct Railway requests, and spoofed `Host`, `X-Real-IP`, or `X-Forwarded-For` cannot produce a trusted Cloudflare identity or expose secret values.

## 3. Admission policy

- [x] 3.1 Create `internal/requestprotection` classification for search, `/dual-mode` fragment, detail, and exempt route families plus canonical-host routing checks; verify table-driven tests cover canonical/deployment hosts, GET/HEAD, ports, host case/trailing dots, health, static, sitemap, robots, admin, and unclassified routes.
- [x] 3.2 Implement bounded per-client rate accounting using only resolved private identities; verify tests cover separate search/fragment/detail limits, window expiry, failed identity resolution, fixed-capacity eviction, address rotation, and absence of raw identity data.
- [x] 3.3 Implement the non-blocking global protected-read capacity gate and idempotent release contract; verify deterministic concurrent tests cover available, exhausted, cancelled, and exactly-once release paths without an unbounded queue.
- [x] 3.4 Implement `off`, `observe`, and `enforce` decisions plus privacy-safe structured fields; verify observe mode updates bounded rate state and records would-be decisions without rejecting or retaining scarce concurrency capacity.

## 4. Request integration

- [x] 4.1 Register thin protected-read middleware using configured public URL, Cloudflare origin authentication, `requesttrust.Resolver`, request-scoped logging, and admission policy; verify HTTP tests return 421 for host mismatch, 403 for missing/invalid origin authentication, 429 for client exhaustion, 503 for capacity exhaustion, and `Retry-After` for admission failures before handler work.
- [x] 4.2 Verify exempt and progressive-enhancement contracts through HTTP tests covering direct Railway `/health`, static assets, sitemap and robots routes, authenticated canonical full-page catalogue requests, and authenticated canonical HTMX fragment requests.
- [x] 4.3 Add minimal plain 403/421/429/503 responses and verify rejection paths do not invoke Templ, expose diagnostics, or produce Sentry server-fault events.
- [x] 4.4 Add structured logging assertions for host, origin-authentication, client-rate, capacity, and observe-mode decisions; verify stable event/profile/status/capacity fields and omission of raw addresses, forwarding headers, limiter keys, secrets, and requested slugs.

## 5. Crawler guidance

- [x] 5.1 Extend application-owned `robots.txt` to retain the canonical sitemap while disallowing `/dual-mode`, `/artworks/results`, and query-string variants without disallowing canonical artist or artwork records; verify the static-handler discovery tests assert the complete crawler contract.
- [x] 5.2 Add bounded public artist/artwork Markdown projections and deterministic renderers; verify focused tests cover canonical URLs, attribution, related links, Markdown escaping, published fields, and exclusion of itinerary, cookie, script, administrative, and private persistence data.
- [x] 5.3 Extend the sitemap generation lifecycle to atomically publish `/llms.txt` and current `/agents/artists/{id}.md` and `/agents/artworks/{id}.md` files under `app.DataDir()`, prune stale records, and log bounded results; verify failed generation leaves the previous publication intact and unavailable/unpublished IDs have no file.
- [x] 5.4 Serve generated agent resources with Markdown content type, canonical link, explicit public cache policy, and no `Set-Cookie`; add canonical `Accept: text/markdown` redirects plus alternate links and verify negotiation occurs before database/view work while ordinary HTML and HTMX remain unchanged.

## 6. Cancellation checkpoints

- [x] 6.1 Add cancellation checkpoints around material stages in artist and artwork search workflows; verify focused tests cancel between stages, observe the original context error, and prove the subsequent repository/render stage is not invoked.
- [ ] 6.2 Add cancellation checkpoints around material stages in artist-detail and artwork-detail workflows; verify focused tests cancel between lookup, projection, related-content, and render stages without recording a Sentry server fault.
- [ ] 6.3 Add cancellation checkpoints around material dual-mode workflow stages; verify focused tests stop subsequent work after cancellation while active full-page and HTMX requests remain unchanged.
- [ ] 6.4 Add request-scoped cancellation telemetry at the shared checkpoint boundary; verify logging tests record route profile and stage without raw client identity or requested slug data.

## 7. Deterministic integration verification

- [ ] 7.1 Run formatting, `go test` for config/requesttrust/requestprotection/handlers/observability, `go vet` on affected packages, and `git diff --check`; correct only failures caused by this change.
- [ ] 7.2 Start the local application with enforcement enabled and run focused Playwright catalogue-search/navigation tests plus authenticated-origin and agent-content HTTP checks; verify normal browser/HTMX interactions succeed, Markdown negotiation resolves to generated content, and rejected requests never reach protected handlers.

## 8. Cloudflare Free configuration

- [ ] 8.1 Verify `beta.wga.hu` resolves through Cloudflare from authoritative, `1.1.1.1`, `8.8.8.8`, and local resolvers and that HTTPS responses carry Cloudflare evidence; document and resolve any stale direct-Railway DNS path before enabling application enforcement.
- [ ] 8.2 Configure Cloudflare SSL/TLS Full (strict), Browser Integrity Check, Always Use HTTPS after canonical verification, and a Request Header Transform Rule that overwrites `X-WGA-Edge-Secret`; verify a canonical request reaches staging with the configured secret while a visitor-supplied value is overwritten.
- [ ] 8.3 Configure the single Free rate-limit rule to exclude verified bots and cover `/artists`, `/artists/*`, `/artworks`, `/artworks/results`, `/dual-mode`, and `/agents/*`, initially blocking above 15 requests per IP per ten seconds for ten seconds; verify Cloudflare Security Events and origin logs show excess requests blocked before origin while verified-bot traffic is excluded.
- [ ] 8.4 Establish a functional baseline, enable Free Bot Fight Mode, and exercise monitoring, accessibility, browser, HTMX, and canonical crawler checks; verify Security Events are reviewed and that disabling Bot Fight Mode independently restores any legitimate traffic it incorrectly blocks.
- [ ] 8.5 Confirm Cloudflare Managed `robots.txt` remains disabled and configure a dedicated Cache Rule only for successful `/agents/*` and `/llms.txt` responses; verify agent resources become edge cache hits while 404, `Set-Cookie`, full-page HTML, and HTMX responses remain ineligible.

## 9. Staged operational rollout

- [ ] 9.1 Read the documentation maintenance sources and document the Cloudflare Free rule expressions, agent Markdown discovery/cache contract, generated-publication recovery, origin-secret setup/rotation, protection modes, threshold tuning, DNS checks, Bot Fight Mode rollback, and application rollback; verify documentation formatting/checks pass without recording the secret.
- [ ] 9.2 Deploy application protection to staging in observe mode and exercise sustained plus burst HTML and agent traffic; verify rejected-request latency stays below 100 ms, repeated Markdown requests become edge cache hits, health remains continuously available, memory remains stable, expected cancellations stay out of Sentry, and recorded CPU/latency evidence supports the selected enforcement thresholds.
- [ ] 9.3 Enable staging enforcement with Cloudflare protections active; verify direct Railway requests fail even with a spoofed canonical Host, legitimate canonical navigation remains functional, Cloudflare and application limits each activate under their intended load, and rollback to observe mode succeeds.
- [ ] 9.4 Rotate the origin secret in staging using the current/next keyring sequence and verify uninterrupted authenticated traffic, rejection of the retired value, and absence of secret material from logs and Sentry.
