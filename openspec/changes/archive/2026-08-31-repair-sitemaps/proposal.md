## Why

The sitemap generator writes files that the application does not consistently expose, and it does not provide the conventional public index or discovery contract expected by crawlers. Generated files are also tied to the process working directory, making availability unreliable across restarts and deployments.

This change restores a reliable, crawler-readable sitemap while providing a browser presentation consistent with the site.

## What Changes

- Publish a canonical sitemap index at `/sitemap.xml`, with matching publicly served child sitemap URLs.
- Store generated sitemap files under the PocketBase data directory so they share the application's persistent storage location.
- Generate a complete sitemap after the application is ready and regenerate it on the existing daily schedule without terminating the application when generation fails.
- Add sitemap discovery through `robots.txt`.
- Apply a same-origin XSL presentation to sitemap XML so browsers render it with the site's existing visual language, while preserving valid crawler-readable XML.
- Add generation, serving, URL-inclusion, and failure-path coverage for the sitemap contract.

## Capabilities

### New Capabilities
- `sitemap-publication`: Generate, publish, discover, and present canonical public XML sitemaps.

### Modified Capabilities

- None.

## Impact

- Affected code: sitemap generation and cron registration, static/public route handling, configuration-derived application data paths, and sitemap tests.
- Affected public endpoints: `/sitemap.xml`, child sitemap paths, `/sitemap.xsl`, and `robots.txt`.
- Affected runtime storage: derived sitemap files under the PocketBase data directory.
- Dependencies: retain the existing sitemap generator dependency; no new third-party dependency is proposed.
