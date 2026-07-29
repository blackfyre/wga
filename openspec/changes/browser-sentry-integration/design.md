## Context

The existing Sentry integration treats `WGA_SENTRY_DSN` as a shared server and browser setting. The two runtimes now use different Sentry projects, so that coupling routes browser events to the server project. The existing browser bundle already reads public configuration from page metadata before loading the main bootstrap and removes sensitive URL and console-breadcrumb data.

## Goals / Non-Goals

**Goals:**

- Configure server and browser Sentry SDKs independently.
- Keep both DSNs optional and preserve the existing disabled behaviour for either runtime.
- Expose only the browser project's public DSN and deployment environment to page code.

**Non-Goals:**

- Add Sentry tracing, replay, release tracking, or new event types.
- Change the existing event scrubbing policy or server error-capture behaviour.
- Commit the supplied DSN to source control.

## Decisions

### Use two explicit optional environment settings

Keep `WGA_SENTRY_DSN` as the server setting and add `WGA_SENTRY_BROWSER_DSN` for the browser project. Both values are parsed and validated through `internal/config`, and the sample environment file documents their separate purposes. Explicit settings make the deployment contract clear and avoid an unnecessary runtime-to-DSN mapping abstraction.

### Decouple page configuration from server SDK initialisation

Build browser page configuration directly from the browser DSN and WGA environment, even when server monitoring is disabled. Server SDK initialisation continues to depend only on the server DSN. This allows either project to be enabled independently and prevents an absent server DSN from disabling browser reporting.

### Continue runtime metadata delivery for the browser DSN

Supply the browser DSN and environment through the existing rendered metadata consumed by the npm-bundled `@sentry/browser` entry point. The server DSN is never included in page markup or static assets. This preserves per-environment deployment configuration without rebuilding the frontend bundle.

## Risks / Trade-offs

- [A deployment assigns the DSNs to the wrong settings] → Document each setting and manually confirm the server and browser test events appear in their intended projects before production use.
- [A browser DSN is treated as confidential] → Treat it as public client configuration, while retaining validation that rejects secret-bearing DSNs and avoiding logs or source-controlled values.
- [One DSN is omitted] → Keep its runtime disabled while allowing the configured runtime and the rest of the application to start normally.

## Migration Plan

1. Deploy the configuration change with both environment variables empty or with the existing server DSN unchanged.
2. Set `WGA_SENTRY_BROWSER_DSN` to the supplied browser-project DSN in non-production deployment configuration.
3. Confirm server and browser `/sentry-test` events arrive in their separate projects with the expected environment.
4. Roll back browser monitoring by removing `WGA_SENTRY_BROWSER_DSN`; remove the release only if the configuration change itself must be reverted.

## Open Questions

None.
