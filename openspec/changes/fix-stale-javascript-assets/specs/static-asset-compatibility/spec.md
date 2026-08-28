## Purpose

Ensures a deployed browser can move between releases without a cached entry module loading unavailable JavaScript chunks.

## ADDED Requirements

### Requirement: Deployment-compatible JavaScript loading
The application SHALL serve the browser JavaScript entry module with revalidation semantics and content-hashed JavaScript chunks with immutable caching semantics.

#### Scenario: Browser has a stale entry module
- **WHEN** a browser loads the application after a deployment with a previously cached entry module
- **THEN** it revalidates the entry module before loading dynamic chunks

#### Scenario: Browser loads a current hashed chunk
- **WHEN** a browser requests a JavaScript chunk referenced by the current entry module
- **THEN** the response permits immutable caching

### Requirement: Dynamic module recovery
The browser SHALL recover once from a dynamic-module load failure by reloading the page, and SHALL present an actionable failure state if the failure persists.

#### Scenario: First dynamic-module failure
- **WHEN** a dynamic JavaScript module fails to load
- **THEN** the browser reloads the page once without discarding the current URL

#### Scenario: Persistent dynamic-module failure
- **WHEN** the same dynamic-module load failure occurs after recovery
- **THEN** the browser presents a reload action rather than repeatedly reloading
