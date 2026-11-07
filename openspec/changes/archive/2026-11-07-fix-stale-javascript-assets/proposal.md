## Why

After a deployment, a browser with a cached `app.js` can request a removed content-hashed dynamic chunk and leave the application unusable. Sentry recorded this as `WGA-BROWSER-4`.

## What Changes

- Ensure the browser receives a compatible JavaScript entry point across deployments.
- Preserve immutable caching for content-hashed assets while preventing stale entry-point references.
- Provide a recoverable user experience when a dynamic module cannot be fetched.

## Capabilities

### New Capabilities
- `static-asset-compatibility`: deployment-safe browser asset loading and recovery behaviour.

### Modified Capabilities

None.

## Impact

- Static asset serving, frontend build output, browser bootstrap, and deployment cache headers.
- No API, data, or authentication changes.
