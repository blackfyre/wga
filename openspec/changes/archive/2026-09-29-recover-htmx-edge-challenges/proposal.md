# Proposal

## Why

When Cloudflare answers an enhanced (HTMX) request with a managed challenge (`403`, `cf-mitigated: challenge`), the browser cannot solve it: a challenge needs a full page load to run and set clearance. The interaction silently does nothing. This happens today with the top-navigation link to `/artworks` on `beta.wga.hu`, while typing the URL works.

## What Changes

- When an enhanced `GET` request receives an edge challenge response, the browser discards the response and loads the requested URL as a full page. There the challenge can be solved and the destination rendered.
- Challenged non-`GET` requests are left unchanged, because repeating them as a navigation could lose or duplicate submitted data.
- Out of scope: changing Cloudflare rules; the rule that challenges `/artworks` has to be reviewed separately.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: enhanced navigation recovers from edge challenges by falling back to a full page load.

## Impact

- New `resources/js/edge-challenge.ts`, initialised once from `bootstrap.ts`.
- A JS unit test and a Playwright check.
- No server, route, or configuration changes.
