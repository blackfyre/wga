# Proposal

## Why

Protected public reads share one global concurrent-work gate. A distributed flood on `/dual-mode` kept all eight slots busy, so search and detail visitors were also rejected with 503 even though their routes were idle. Per-client limits cannot help against rotating residential addresses, so the useful containment is to stop one route family's load from consuming the others' capacity.

## What Changes

- Give each protected route profile (search, fragment, detail) its own concurrent-work gate, sized by `WGA_PUBLIC_READ_MAX_CONCURRENT`.
- Observe mode tracks would-be capacity per profile.
- Capacity gauges carry the bounded route-profile attribute.

## Capabilities

### Modified Capabilities

- `public-request-protection`: admission combines per-client rate limits with a concurrent-work limit per route profile instead of one global limit.

## Impact

`internal/requestprotection` (policy, telemetry), its tests and the handler middleware tests, `.env.example`, and `docs/bot-protection-runbook.md`. The origin may now run up to three times the configured value of protected work concurrently.
