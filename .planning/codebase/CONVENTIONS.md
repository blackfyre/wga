# Conventions

## Go Style

- Package names are lower-case and follow directory names.
- Feature packages commonly expose `RegisterHandlers(app *pocketbase.PocketBase)`.
- Most files use explicit, direct control flow instead of heavy abstraction.
- Error handling is inline and immediate, with logging through `app.Logger()` and early returns.
- Request handlers often return shared helper responses such as `utils.ServerFaultError(c)` and `utils.BadRequestError(c)`.

## File and Package Patterns

- Many packages use `main.go` as the primary file regardless of package name, for example `internal/handlers/landing/main.go`.
- Route feature areas are grouped into small packages under `internal/handlers/`.
- Utilities are spread across focused files rather than a single kitchen-sink helper package, although `internal/utils/main.go` is still broad.
- Migrations are append-only and timestamp-prefixed.

## Template Conventions

- Templ source files live next to generated Go output.
- Page rendering usually happens through DTO/page structs and `Render(ctx, &buf)`.
- Template metadata is set through context decoration helpers in `internal/assets/templ/utils/`.
- Wrapped page helpers indicate layout composition, for example `pages.HomePageWrapped(...)`.

## Frontend Conventions

- Biome enforces tab indentation and double quotes in JS/TS files.
- Frontend assets are built from `resources/` into committed files under `internal/assets/public/`.
- The client-side layer appears intentionally light; HTMX-style server interactions are preferred over a large SPA state layer.

## Testing Conventions

- Newer Go tests use table-driven subtests in places like `internal/handlers/dual/main_test.go`.
- Older tests still use simpler one-off assertions and direct `t.Errorf` / `t.Fatalf`.
- Playwright tests use direct locators and real-page flows rather than a page-object abstraction.

## Data and Configuration Conventions

- Environment variables use the `WGA_` prefix consistently.
- Collection names and similar repeated identifiers are centralized in `internal/constants/`.
- Cached values use simple string keys in the PocketBase app store with explicit TTL metadata.

## Logging and Observability

- PocketBase logger is used directly instead of a separate logging abstraction.
- Logging is descriptive but not deeply structured; messages often include `"error", err.Error()` pairs.
- Debug logs are present for handler, hook, and cron registration.

## Notable Inconsistencies

- Phase 2 corrected the contributor-facing docs to use `cmd/wga`, `internal/*`, `resources/*`, and `playwright-tests/*`; the remaining risk is reintroducing stale path or command terminology in later docs.
- `package.json` still declares `lint-staged` with `prettier`, while the repo guidance centers on Biome.
- Some tests are modernized while others remain ad hoc, so testing style is not fully uniform yet.
