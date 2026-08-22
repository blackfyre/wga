## Context

The regenerated producer and live WGA databases both contain `scripts:header` with the preserved ID `779bca08400c869`. The producer record now contains an intentionally supplied trusted fragment, while the live WGA record remains empty. The WGA `strings` collection exposes `name` and editor `content` fields and has null list, view, create, update, and delete API rules, making it superuser-only.

Full pages and shared error pages render Templ components with a context derived from the request. No request-time `strings` reader exists. The shared `LayoutBase` owns the document `<head>`, while HTMX responses render fragments without requiring document-head content.

Production verification after the initial implementation found that this assumption was not uniformly true at the committed baseline: several full-page handlers rendered from `context.Background()` and therefore discarded values prepared by request middleware. Only handlers already deriving their render context from the request displayed the trusted fragment.

Templ escapes ordinary string expressions. Rendering operator-supplied script elements therefore requires an explicit use of Templ's trusted raw-HTML boundary. See `specs/database-managed-head-markup/spec.md` for the required behaviour.

## Goals / Non-Goals

**Goals:**

- Keep PocketBase access in the handler adapter layer and pass an immutable string through request context to Templ.
- Read the current value only for public full-document requests, not for assets, PocketBase APIs, or HTMX fragments.
- Make the raw rendering boundary conspicuous in names, tests, and code review.
- Keep optional-script failures from taking down otherwise renderable pages.

**Non-Goals:**

- Sanitising, parsing, rewriting, validating, or allow-listing the trusted fragment.
- Populating either production database, changing collection rules, or enforcing a new uniqueness constraint.
- Adding caching, update hooks, footer injection, a management UI, CSP changes, or a new public API.

## Decisions

### 1. Prepare trusted head markup in a global handler middleware

Register one middleware before feature routes. For eligible public document requests, it looks up `strings.name = "scripts:header"`, obtains `content`, and replaces the request context with a context carrying that value before continuing.

Eligibility is limited to non-HTMX `GET` or `HEAD` requests that negotiate HTML and do not target technical route boundaries such as `/api`, `/_`, `/assets`, `/sitemap`, or the visual reference route. This avoids a database lookup for embedded assets and API traffic while still covering normal pages and shared public error documents.

The lookup runs once per eligible request. This keeps operator changes visible on the next document request and avoids cache invalidation machinery. The table has seven rows, so a single exact-name lookup is preferable to a process-wide cache and record-update hooks.

Alternative considered: load once at server start. Rejected because changes made through the superuser interface would require a restart. Alternative considered: let the template perform a lazy lookup. Rejected because rendering must remain free of hidden database side effects.

### 2. Use a typed context contract dedicated to trusted markup

Add a private typed context key plus narrowly named setter and getter helpers in the Templ utility package. Do not reuse the exported generic `ContextKey` or `DecorateContext`, because an ordinary string key obscures the executable-content trust boundary and can collide with unrelated values.

The context contains only an immutable string. Missing values resolve to an empty string.

Alternative considered: add a parameter through every page and layout component. Rejected because it would propagate infrastructure content through unrelated page DTOs and call sites.

### 3. Render once at the end of the shared document head

`LayoutBase` obtains the trusted fragment from context and, only when non-empty, renders it once immediately before `</head>` using Templ's raw HTML component API. Ordinary Templ expressions remain escaped everywhere else.

The function and local names must include `Trusted` or equivalent wording, and the raw call receives only the value obtained through this dedicated context contract. This is an intentional stored-script execution boundary, not a general rich-text helper.

Alternative considered: use the existing `html/template` `safeHTML` helper. Rejected because that helper belongs to the legacy template function map and would blur the Templ-specific boundary.

### 4. Fail open only for the optional fragment

A missing row and empty content are normal and produce no fragment. Other lookup failures are logged with `logging.RequestLogger`, an error type, and redacted detail; record content is never logged. The middleware then continues without trusted markup.

This fail-open choice applies only to the optional header fragment. It does not change error handling for the requested page's own data or rendering.

### 5. Preserve request context in every full-document handler

Every handler that renders a full document through `LayoutBase` must derive its Templ context from the current HTTP request before adding page metadata. This preserves trusted head markup and other request-scoped values without coupling feature handlers directly to the strings collection.

The repair updates only context initialisation sites that currently start from `context.Background()`. Fragment-only rendering paths may continue to use a background context when they do not render the shared document layout.

Alternative considered: store trusted markup in process-global state so layouts can recover it after context loss. Rejected because it introduces cross-request mutable state, weakens request isolation, and hides the existing handler contract violation.

## Risks / Trade-offs

- [A compromised superuser or producer input can execute code on every full page] → Keep all collection API rules superuser-only, expose no new write path, name the trust boundary explicitly, and cover raw rendering with focused tests.
- [Inline or third-party scripts can weaken privacy, performance, or browser security] → Content governance remains an operator responsibility; this change does not bypass browser CSP or grant server-side capabilities.
- [One extra lookup occurs for each full document] → Restrict middleware eligibility to document requests and avoid lookups for assets, APIs, and fragments.
- [The WGA schema does not enforce `strings.name` uniqueness] → Preserve current scope and use the existing exact-name first-record lookup; producer data enforces uniqueness. Treat duplicate operator-created rows as an operational data defect rather than silently expanding this change into a migration.
- [Header markup failure is intentionally non-fatal] → Emit request-scoped redacted diagnostics so operators can distinguish absence from database failure.

## Operational Decomposition

One bounded implementation workstream owns the shared handler middleware, typed Templ context contract, `LayoutBase`, generated Templ output, and focused tests. These files share one contract and SHALL be changed serially by one implementation owner; no parallel writers are required.

Independent assurance reviews the stored-script boundary and verifies objective evidence after implementation. No database, producer, configuration, or frontend-bundle workstream is authorised.

## Verification

- Focused Go tests prove request eligibility, successful context decoration, missing/empty/error behaviour, redacted request-scoped logging, and no lookup for technical or HTMX requests.
- Layout render tests prove exact one-time raw placement before `</head>`, ordinary absence behaviour, and that no fragment component gains the markup.
- Router-level integration tests prove representative full-page handlers preserve middleware-provided trusted markup through their real render path.
- `templ generate` regenerates ignored Go output from the edited Templ source.
- `go test` runs the focused handler and layout packages, followed by `go vet ./...` and `go test ./...` if the focused evidence passes.
- A local runtime check temporarily supplies benign marker script content in disposable test data and confirms it appears in a full page head but not an HTMX fragment; production seed data remains unchanged.

## Migration Plan

Deploy the code without changing data. Each deployment renders the `scripts:header` content already present in its own database; this change does not synchronise producer and live records.

Rollback removes the middleware/context/layout integration. The existing database record remains compatible and requires no data rollback.
