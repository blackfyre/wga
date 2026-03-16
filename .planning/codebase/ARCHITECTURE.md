# Architecture

## High-Level Shape

- The application is a server-rendered Go web app built on PocketBase.
- `cmd/wga/main.go` creates the PocketBase app, registers handlers, hooks, cron jobs, migrations, and extra CLI commands, then starts the server.
- There is no separate API server and SPA; HTML is rendered on the backend and enhanced with small frontend bundles.

## Request Flow

1. PocketBase boots in `cmd/wga/main.go`.
2. `internal/handlers/main.go` wires feature modules into the shared app instance.
3. Each feature package binds routes during `app.OnServe()`.
4. Handlers read records or repository results from PocketBase.
5. Templ page/components render HTML, usually through `bytes.Buffer`.
6. The handler returns HTML directly or HTMX-friendly partial responses.

## Application Layers

- `internal/handlers/` contains route registration and request orchestration.
- `internal/repositories/` wraps repeated data access patterns for some domains.
- `internal/utils/` contains cross-cutting helpers for caching, URL generation, pagination, JSON-LD, sitemap logic, and general helpers.
- `internal/assets/templ/` holds view definitions and generated Go render functions.
- `internal/assets/views/` holds non-Templ rendered fragments such as email bodies.
- `internal/crontab/` contains background jobs registered on PocketBase cron.
- `internal/hooks/` contains PocketBase event hooks.
- `internal/migrations/` seeds/configures PocketBase collections and settings.

## Rendering Pattern

- Handlers typically build a DTO or page struct, render a Templ component/page into a buffer, and send the result with `c.HTML(...)`.
- Layout composition is handled in Templ wrappers such as page `...Wrapped(...)` functions generated from `internal/assets/templ/layouts/`.
- Metadata is injected into `context.Context` via helpers in `internal/assets/templ/utils/`.

## Data Access Pattern

- Simple reads often use PocketBase convenience methods directly from handlers.
- More structured or repeated queries are moved into repository types, for example `internal/repositories/LandingRepository`.
- Some queries use SQL through `app.DB().NewQuery(...)` rather than only collection helpers.
- In-memory app store caching is used for hot values such as landing page counts and welcome text.

## Routing Model

- Route registration is split by feature package: artists, artworks, landing, postcards, guestbook, feedback, static pages, contributors, inspire, and dual mode.
- Dual mode is a notable feature-specific route family that composes two page panes in one UI.
- Static and dynamic page generation coexist in the same PocketBase routing surface.

## Background and Side Effects

- Cron-driven postcard sending reads queued records and sends mail.
- Sitemap generation is exposed both as a cron task and a Cobra command.
- File-download hook support exists in `internal/hooks/files.go`, though the current implementation mostly logs requests.

## Notable Architectural Characteristics

- The app is monolithic and cohesive rather than service-oriented.
- Domain logic remains close to handlers; there is no deep service layer abstraction.
- Template generation and static asset compilation are part of normal development flow, so generated files are part of the repo shape.
- PocketBase is both framework and platform dependency, which keeps the app compact but increases coupling to PocketBase concepts.
