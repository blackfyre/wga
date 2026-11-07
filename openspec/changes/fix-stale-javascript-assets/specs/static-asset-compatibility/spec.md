## Purpose

Ensures a deployed browser can move between releases without a cached entry module loading unavailable JavaScript chunks.

## ADDED Requirements

### Requirement: Deployment-compatible JavaScript loading

The application SHALL reference the browser JavaScript entry module through a URL versioned by the entry module's content, SHALL serve the entry module with revalidation semantics, and SHALL serve existing content-hashed JavaScript chunks with immutable caching semantics. A response for a JavaScript asset that does not exist SHALL NOT permit immutable caching.

#### Scenario: Browser has a stale entry module

- **WHEN** a browser loads the application after a deployment that changed the entry module, with the previous entry module still cached
- **THEN** the page references the entry module under a URL that differs from the previous release's, so the browser does not reuse the cached entry module

#### Scenario: Intermediary overrides the entry cache policy

- **WHEN** a proxy or CDN replaces the entry module's revalidation header with a browser cache lifetime
- **THEN** a later release that changes the entry module still references it under a new URL

#### Scenario: Browser loads a current hashed chunk

- **WHEN** a browser requests a JavaScript chunk referenced by the current entry module
- **THEN** the response permits immutable caching

#### Scenario: Browser requests a removed chunk

- **WHEN** a browser requests a JavaScript chunk that the current release does not serve
- **THEN** the not-found response does not permit immutable caching

### Requirement: Dynamic module recovery

The browser SHALL recover once from a dynamic-module load failure by reloading the page, and SHALL present an actionable failure state if the failure persists.

#### Scenario: First dynamic-module failure

- **WHEN** a dynamic JavaScript module fails to load
- **THEN** the browser reloads the page once without discarding the current URL

#### Scenario: Persistent dynamic-module failure

- **WHEN** the same dynamic-module load failure occurs after recovery
- **THEN** the browser presents a reload action rather than repeatedly reloading
