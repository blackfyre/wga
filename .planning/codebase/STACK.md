# Stack

## Runtime and Languages

- Go `1.25` is the primary backend/runtime language, declared in `go.mod`.
- TypeScript powers the small frontend bundle from `resources/js/app.ts`.
- PostCSS processes the stylesheet entrypoint at `resources/css/style.pcss`.
- Templ templates in `internal/assets/templ/` generate Go render functions alongside the `.templ` sources.
- PocketBase is the application host, database layer, auth/admin surface, cron scheduler, mail client, and file serving layer.

## Backend Frameworks and Libraries

- `github.com/pocketbase/pocketbase` is the application foundation and is bootstrapped in `cmd/wga/main.go`.
- `github.com/labstack/echo/v5` is pulled transitively through PocketBase request handling and route registration.
- `github.com/a-h/templ` is used for component/layout/page rendering under `internal/assets/templ/`.
- `github.com/spf13/cobra` adds CLI commands such as `generate-sitemap` and `generate-music-urls`.
- `github.com/microcosm-cc/bluemonday` is used when registering postcard handlers for sanitization policy wiring.
- `github.com/pocketbase/dbx` is used indirectly for SQL-style querying in repository code such as `internal/repositories/landing.go`.

## Frontend Tooling

- `bun` is the JavaScript package manager and build runner.
- `bun build` bundles `resources/js/app.ts` into `internal/assets/public/js/`.
- `postcss-cli` compiles `resources/css/style.pcss` into `internal/assets/public/css/style.css`.
- `@biomejs/biome` is configured for formatting/linting frontend code in `biome.json`.
- Installed UI libraries include `htmx.org`, `choices.js`, `viewerjs`, `trix`, `animate.css`, `daisyui`, and Tailwind-related packages.

## Development Environment

- `devenv.nix` is the main local environment definition.
- `devenv` provisions Go, Bun, `templ`, `air`, MailHog, and MinIO.
- Common scripts live in `devenv.nix` as `app:build`, `app:run`, `app:tidy`, `app:generate-templates`, and `code:run`.
- The current repo layout uses `cmd/wga/main.go` as the actual entrypoint, not a top-level `main.go`.

## Build Outputs and Generated Artifacts

- Built frontend assets are committed under `internal/assets/public/`.
- Generated Templ Go files sit next to their source templates, for example `internal/assets/templ/pages/home_templ.go`.
- Email and HTML fragment outputs also exist under `internal/assets/views/`.
- PocketBase runtime data is expected under `./wga_data`.

## Data and Content Inputs

- Reference content is shipped in `internal/assets/reference/` as JSON and compressed `.zst` datasets.
- PocketBase collection schema/data bootstrapping lives in `internal/migrations/`.
- Static media such as icons, logos, and audio are served from `internal/assets/public/`.

## Commands Worth Knowing

- `devenv up` starts the local dev processes and services.
- `devenv shell` enters the shared toolchain.
- `go test ./... -cover` runs backend tests.
- `bunx playwright test` runs browser tests.
- `templ generate` refreshes generated template code.
