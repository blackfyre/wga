## 1. Request Context Preparation

- [x] 1.1 Add the typed trusted-head-markup context setter and getter, and verify focused utility tests return the supplied immutable value and default to empty.
- [x] 1.2 Implement the public full-document request eligibility predicate, and verify table-driven tests exclude HTMX, assets, PocketBase APIs, sitemap, and visual-reference traffic.
- [x] 1.3 Add and register the header-markup lookup middleware before feature routes, and verify focused handler tests cover non-empty, empty, missing, changed, and failed lookups with request-scoped redacted logging.

## 2. Shared Layout Rendering

- [x] 2.1 Render non-empty trusted head markup once immediately before `</head>` in `LayoutBase` using Templ's explicit raw HTML API, and verify layout tests cover exact raw output, placement, absence, and ordinary escaped expressions.
- [x] 2.2 Run `templ generate` and verify the focused Templ utility, handler middleware, and layout test packages pass without editing generated files directly.

## 3. Integration and Assurance

- [x] 3.1 Run `go vet ./...` and `go test ./...`, and record passing output or resolve failures caused by this change.
- [x] 3.2 Using disposable test data only, verify a benign marker appears in a normal full-page `<head>`, changes on the next request, and remains absent from an HTMX fragment; also verify the producer and live production records remain byte-for-byte unchanged from their pre-check values.
- [x] 3.3 Independently review the stored-script execution boundary, collection API rules, failure logging, and OpenSpec conformance, and resolve any material finding before marking the change complete.

## 4. Production Regression Repair

- [x] 4.1 Replace background-rooted Templ contexts in every full-document handler with contexts derived from the current request, while leaving fragment-only paths unchanged; verify a source audit finds no remaining full-layout handler that discards request context.
- [x] 4.2 Add router-level regression coverage with populated `scripts:header` data and verify representative full-page routes render the fragment once before `</head>` while HTMX fragments remain unchanged.
- [x] 4.3 Validate the repair from a clean committed baseline plus only the corrective patch using focused tests, `go vet ./...`, and `go test ./...`.
- [x] 4.4 Verify a fresh seeded runtime preserves the producer row byte-for-byte and renders it on the beta homepage path; obtain independent acceptance review before completing the change.
