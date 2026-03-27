# CLAUDE.md — Web Gallery of Art

## Project Overview

Web Gallery of Art (WGA) is a Go web application for browsing European fine arts from the 3rd to 19th centuries. Built on PocketBase (SQLite-backed BaaS), htmx for interactivity, and Templ for server-side HTML rendering. The frontend uses TailwindCSS v4 with DaisyUI components.

## Tech Stack

- **Go 1.25+** with PocketBase v0.31
- **Templ** for type-safe HTML templates
- **htmx 2.0** for partial page updates
- **TailwindCSS v4** + **DaisyUI v5** for styling
- **Bun** for JS/CSS build tooling
- **PostCSS** pipeline with autoprefixer and cssnano
- **PocketBase** as the database (SQLite), auth, file storage, and admin UI
- **MinIO** (S3-compatible) for file storage in development
- **Mailpit** for email testing in development

## Project Structure

```
cmd/wga/main.go              # Application entry point
internal/
  assets/
    public/                   # Built CSS/JS/images (served at /assets/)
    reference/                # Seed data (JSON, compressed .json.zst)
    templ/
      components/             # Reusable UI components (nav, footer, image, dialog)
      dto/                    # Data Transfer Objects (Go structs for templates)
      error_pages/            # 400, 404, 500 error pages
      layouts/                # Base layouts (LayoutBase, LayoutMain, LayoutSlim)
      pages/                  # Full page templates
      utils/                  # Context decoration, template helpers
  constants/                  # Collection name constants
  crontab/                    # Scheduled jobs (postcards, sitemap)
  errs/                       # Custom error variables
  handlers/                   # HTTP handlers organized by feature
  hooks/                      # PocketBase lifecycle hooks
  migrations/                 # Database schema + seed data
  repositories/               # Data access with caching
  utils/                      # Helpers (cache, pagination, URL, htmx, zstd)
  validation/                 # Input validation (honeypot, message, reCAPTCHA)
resources/
  css/style.pcss              # PostCSS source (Tailwind + DaisyUI)
  js/app.ts                   # TypeScript entry point
playwright-tests/             # E2E browser tests
```

## Build & Run

### Prerequisites

- Go 1.25+, Bun, Templ (`go install github.com/a-h/templ/cmd/templ@latest`)
- Podman/Docker for MinIO and Mailpit services

### Commands

```bash
# Start supporting services
podman-compose up -d

# Install JS deps + build assets
bun install && bun run build

# Generate templ templates
templ generate

# Build Go binary
mkdir -p dist && go build -v -o dist/wga ./cmd/wga

# Run (from dist/ directory, where .env lives)
cd dist && ./wga serve --dev

# One-liner rebuild
bun run build && templ generate && go build -v -o dist/wga ./cmd/wga
```

### Running Tests

```bash
go test ./... -cover              # Go unit tests
bunx playwright test              # E2E tests
bunx playwright test --headed     # E2E with browser visible
```

## Code Conventions

### Go

- **Formatting**: `go fmt ./...` — standard Go formatting
- **Package names**: lowercase, matching folder name (e.g., `handlers/dual` → `package dual`)
- **Imports**: stdlib first, then third-party, then internal packages. Alias `tmplUtils` for `internal/assets/templ/utils`
- **Error handling**: Use `utils.ServerFaultError(c)`, `utils.BadRequestError(c)`, `utils.NotFoundError(c)` for HTTP errors. Log via `app.Logger().Error/Warn/Debug()`
- **Custom errors**: Define in `internal/errs/` as `var Err* = errors.New(...)`
- **Collections**: Always reference via `constants.Collection*` constants, never magic strings
- **Tests**: Co-located `_test.go` files, table-driven subtests with `t.Run()`

### Handler Pattern

Each feature lives in its own package under `internal/handlers/`:

```go
package myfeature

func RegisterHandlers(app *pocketbase.PocketBase) {
    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        se.Router.GET("/path", func(c *core.RequestEvent) error {
            return myHandler(app, c)
        })
        return se.Next()
    })
}
```

Then register in `internal/handlers/main.go`:
```go
myfeature.RegisterHandlers(app)
```

### Templ Template Pattern

Templates follow a **DTO → Page → Block** pattern:

1. **DTO** in `dto/` package — plain Go struct
2. **Page** template wraps layout + block: `templ MyPage(v MyDTO) { @layouts.LayoutMain() { @MyBlock(v) } }`
3. **Block** template renders the actual content (reusable in htmx partial responses)

### Context Decoration (SEO/Meta)

```go
ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Page Title")
ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Description")
ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)
```

Title and Description automatically sync to OG and Twitter meta tags.

### HTMX Response Pattern

```go
// Set browser URL for htmx navigation
c.Response.Header().Set("HX-Push-Url", pushUrl)

// Return partial for htmx, full page for normal requests
var buf bytes.Buffer
if utils.IsHtmxRequest(c) {
    pages.MyBlock(content).Render(ctx, &buf)
} else {
    pages.MyPage(content).Render(ctx, &buf)
}
return c.HTML(200, buf.String())
```

### Caching Pattern

```go
if cached, ok := utils.GetCachedValue[MyType](app, "cache-key"); ok {
    return cached, nil
}
// ... fetch data ...
utils.SetCachedValue(app, "cache-key", data, 6*time.Hour)
```

### Frontend

- **JS**: TypeScript, global `window.wga` namespace with nested objects (`wga.dialog`, `wga.dual`, `wga.music`)
- **CSS**: DaisyUI components (`btn`, `card`, `alert`, etc.) + Tailwind utilities. Custom themes defined in `style.pcss`
- **Formatting**: Biome — tab indentation, double quotes, sorted imports. Run `bunx @biomejs/biome format .`
- **Templ files**: kebab-case filenames matching route fragments

### Commit Messages

Imperative mood, 50-72 char subject. Conventional prefixes: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `build:`. Clean up WIP commits via rebase before PR.

## Architecture Notes

- **Database**: PocketBase wraps SQLite. Migrations create collections (tables) with typed fields. Seed data loaded from `internal/assets/reference/` (JSON or zstd-compressed JSON)
- **File storage**: S3-compatible (MinIO in dev, real S3 in production). URLs generated via `utils/url.GenerateFileUrl()` and `GenerateThumbUrl()`
- **Admin UI**: PocketBase admin at `/_/` (auto-configured in dev mode)
- **Cron jobs**: Registered in `internal/crontab/` via `app.Cron().MustAdd()`
- **Repository pattern**: Used for external data (GitHub API) with caching + file fallback. See `internal/repositories/contributors.go`

## CI/CD

- **PR validation**: Conventional commit check, `go vet`, `go test`
- **Release**: Triggered on `v*.*.*` tags. GoReleaser builds Linux amd64 binary
- **E2E**: Playwright tests in separate workflow
- **Deployment**: Fly.io (`fly.toml`) or SSH deploy to UAT

## Key URLs (Development)

- App: http://localhost:8090
- PocketBase Admin: http://localhost:8090/_/
- MinIO Console: http://localhost:9001
- Mailpit: http://localhost:8025
