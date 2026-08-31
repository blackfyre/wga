## Context

See `proposal.md` for motivation and `specs/sitemap-publication/spec.md` for the observable contract. The existing generator writes relative to the process working directory, emits child-map paths that do not match the registered static path, and has no canonical index or discovery endpoint. The application uses a PocketBase data directory that is the suitable persistent location for rebuildable generated files.

## Goals / Non-Goals

**Goals:**
- Publish one canonical, crawler-readable index and matching child maps from application-owned persistent storage.
- Preserve the last complete sitemap set across failed regeneration.
- Make maps discoverable and browser-readable without weakening the XML contract.
- Cover the full generator-to-route contract with focused automated tests.

**Non-Goals:**
- Adding image, video, news, language-alternate, or other specialised sitemap extensions.
- Expanding the sitemap beyond published artist and artwork pages.
- Providing shared sitemap storage or scheduler coordination for a multi-instance deployment.
- Retaining legacy sitemap endpoint aliases.

## Decisions

### Publish an uncompressed canonical XML set

The index will be named and served as `/sitemap.xml`; child files will be served below `/sitemap/`. The index and children will be written as ordinary XML rather than gzip-only files so the canonical URLs are directly browser-readable and simple for crawlers to fetch.

The existing sitemap library will remain responsible for XML URL-set/index formation and splitting. Generated names and the child-map public prefix will be configured as one contract. Replacing the library is rejected because the repair does not require new sitemap formats.

### Place derived output below the PocketBase data directory

The sitemap workflow will derive its output directory from `app.DataDir()` and use a dedicated `sitemaps` child directory. This removes working-directory dependence and aligns sitemap persistence with the application's configured data volume.

The data directory is selected over embedded public assets because sitemap content is runtime-derived. The set remains rebuildable and does not require a schema migration.

### Publish staged output atomically

Each run will write and validate a complete sitemap set in a staging sibling of the public directory, then replace the public set only after every expected file exists and the index references only that set. The workflow will serialise local runs so startup, cron, and manual generation cannot publish concurrently. Failed staged output will be discarded and the current public directory preserved.

In-place writes are rejected because public crawlers can otherwise observe partial XML. A cross-instance lock is deliberately out of scope because this change is scoped to the current single persistent application data volume.

### Keep public serving as explicit routes

The server will provide a canonical index route, a named `{path...}` child-file route, an XSL route, and a plain-text `robots.txt` route. The child-file route will be limited to the generated sitemap directory; filesystem paths will be normalised and traversal rejected by the static serving boundary.

The XSL is a presentation adjunct: generated XML will carry a same-origin stylesheet processing instruction, while the XSL transforms it for browsers and links the current compiled site stylesheet. The XML remains complete and valid without either the XSL or CSS. A separate HTML sitemap page is rejected because it would not apply browser styling to the XML response requested by the user.

### Establish generation lifecycle and failure semantics

Register sitemap generation after application initialisation so it can query migrated data, attempt one initial run, then retain the daily cron cadence. Generator errors will be returned to the caller and logged with a stable outcome event, published URL count, and elapsed time; they will never call process-terminating logging functions.

The application will retain the existing manual generation command as an operator recovery mechanism, but it will use the same workflow and destination as startup and cron execution.

## Operational Decomposition

1. **Sitemap workflow and persistence** — Area: sitemap package, cron registration, and command entrypoint. Owns data selection, canonical URLs, staging, publication, cleanup, metrics/logging, and error propagation.
2. **Public delivery and discovery** — Area: static/public route registration and generated presentation asset. Owns index/child/XSL/robots responses and the response path contract.
3. **Verification** — Area: focused sitemap and static-handler tests. Owns seeded-record generation, XML parsing, endpoint retrieval, failure retention, and browser presentation coverage.

The sitemap workflow and public delivery share endpoint and filename conventions, so their implementation must be serialised under one primary owner. Verification follows both workstreams. No database migration or third-party dependency workstream is required.

## Risks / Trade-offs

- **Generated files are unavailable if the data volume is lost** → Files are derived and regenerated at startup; deployment documentation will identify the directory as rebuildable.
- **A second application replica uses another local data volume** → This design supports one persistent application data volume only; shared storage or a single scheduler is required before horizontal scaling.
- **The current stylesheet asset path changes** → Render the XSL stylesheet response through the existing asset URL helper and cover the generated asset reference in a focused test.
- **Large catalogues increase generation time and memory use** → Preserve sitemap splitting, log duration/counts, and defer streaming database pagination until measured scale requires it.
- **Malformed records cause invalid URLs** → Exclude them from publication and log the exclusion rather than publishing an invalid entry or aborting the complete set.

## Migration Plan

1. Deploy the new routes and generation workflow.
2. On first ready application start, generate the initial data-directory sitemap set and make `/sitemap.xml` discoverable through `robots.txt`.
3. Verify the canonical index and each child sitemap from the deployed public origin.
4. If rollback is needed, remove the new routes/workflow; generated data-directory files are inert derived data and can remain or be removed manually.
