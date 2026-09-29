# Tasks

## 1. Challenge fallback

- [x] 1.1 Add `resources/js/edge-challenge.ts`: an idempotent `htmx:beforeSwap` listener that, for responses carrying `cf-mitigated: challenge`, prevents the swap and, for `GET` requests only, assigns the request URL to `window.location`. Initialise it from `bootstrap.ts`, and cover the decision in `resources/js/edge-challenge.test.ts` (challenged GET navigates, including the query; challenged POST does not; plain 403 is untouched). Verify with `bun test resources/js/edge-challenge.test.ts`, `bunx biome check`, and `bun run build`.

## 2. Browser verification

- [x] 2.1 Add `playwright-tests/edge-challenge.spec.ts`. It intercepts the enhanced request made by the top-navigation Artworks link and answers it with a `403` challenge response, then asserts that the browser performs a full-document load of `/artworks` and renders the Artworks page. It also asserts that a plain `403` without the header causes no navigation. Verify with `mise run test:playwright -- playwright-tests/edge-challenge.spec.ts`.
