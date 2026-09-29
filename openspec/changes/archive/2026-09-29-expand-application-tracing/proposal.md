## Why

Most configured request traces contain only the inbound server span, so they confirm that a request occurred but usually do not explain where its time was spent. WGA needs bounded application-level detail on ordinary catalogue requests while preserving the existing privacy, cardinality, and failure-isolation guarantees.

## What Changes

- Add stable child spans around selected material catalogue workflows and repository operations used by ordinary public requests.
- Add bounded cache outcome events to the active request span so cache behaviour is visible without creating a span for every cache access.
- Keep raw SQL, query values, record identifiers, visitor data, URLs, and arbitrary errors out of trace data.
- Keep trivial parsing, formatting, and helper calls uninstrumented, and do not introduce blanket database auto-instrumentation.

## Capabilities

### New Capabilities

- `application-trace-detail`: Defines bounded workflow, repository-operation, and cache-event detail within configured request traces.

### Modified Capabilities

None.

## Impact

- Affects the shared observability package, instrumented cache helpers, and selected catalogue handlers and repositories.
- Extends emitted OTLP trace structure without changing HTTP APIs, persistence, deployment configuration, or dependencies.
- Requires focused trace-shape and privacy regression tests for each newly instrumented path.
