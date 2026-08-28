## 1. Asset cache policy
- [x] 1.1 Serve the JavaScript entry with revalidation headers and hashed chunks with immutable headers; add focused handler tests.

## 2. Browser recovery
- [x] 2.1 Detect dynamic-import load failures, reload once while retaining the URL, and render a manual retry state on recurrence; add browser tests.

## 3. Verification
- [ ] 3.1 Run focused Go and browser tests, `bun run build`, and verify deployed response headers and recovery behaviour.
