## 1. Asset cache policy

- [x] 1.1 Serve the JavaScript entry with revalidation headers and hashed chunks with immutable headers; add focused handler tests.

## 2. Browser recovery

- [x] 2.1 Detect dynamic-import load failures, reload once while retaining the URL, and render a manual retry state on recurrence; add browser tests.

## 3. Verification

- [x] 3.1 Run focused Go and browser tests, `bun run build`, and verify deployed response headers and recovery behaviour.
  - Staging (`beta.wga.hu`, `main` at `0d87c8df`): hashed chunks carry `public, max-age=31536000, immutable`; dynamic-module recovery (reload once, then the retry alert) passes `playwright-tests/static-asset-compatibility.spec.ts` against staging. The entry module reached browsers as `max-age=14400` (Cloudflare Browser Cache TTL) and a missing chunk's 404 was immutable; both are fixed by the versioned entry URL and existence check, verified locally because staging deploys only from `main`.
