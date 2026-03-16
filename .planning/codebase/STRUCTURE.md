# Structure

## Top-Level Layout

- `cmd/wga/` contains the real application entrypoint.
- `internal/` contains the backend application code.
- `resources/` contains frontend source assets.
- `playwright-tests/` contains browser end-to-end tests.
- `dist/` is the expected binary output directory from the build scripts.

## Internal Package Layout

- `internal/handlers/` contains request handlers grouped by feature.
- `internal/repositories/` contains focused data-access helpers.
- `internal/utils/` contains reusable support code.
- `internal/validation/` contains lightweight request validation helpers.
- `internal/errs/` centralizes reusable error values.
- `internal/hooks/` contains PocketBase event bindings.
- `internal/crontab/` contains scheduled jobs.
- `internal/migrations/` contains migration-driven schema and seed logic.
- `internal/constants/` contains collection-name constants and similar shared identifiers.
- `internal/assets/` contains both source templates and committed build outputs.

## View and Asset Layout

- `internal/assets/templ/components/` holds reusable UI pieces.
- `internal/assets/templ/layouts/` holds page wrappers/layout scaffolding.
- `internal/assets/templ/pages/` holds route-level page templates.
- `internal/assets/templ/error_pages/` holds error renderers.
- `internal/assets/templ/dto/` holds template DTO types.
- `internal/assets/views/` holds rendered email/static HTML templates that are not handled through Templ.
- `internal/assets/public/` holds compiled CSS, JS, images, and other served assets.
- `internal/assets/reference/` holds JSON and compressed reference datasets used by the app.

## Frontend Source Layout

- `resources/js/app.ts` is the frontend bundle entrypoint.
- `resources/js/logger.ts` and `resources/js/wga.d.ts` provide supporting frontend utilities/types.
- `resources/css/style.pcss` is the stylesheet entrypoint.
- `resources/mjml/postcard_notification.mjml` is the MJML source for postcard email markup.

## Testing Layout

- Go unit tests are colocated with packages using `_test.go`.
- Playwright specs sit flat under `playwright-tests/`.
- There is no large dedicated integration-test directory beyond Playwright.

## Naming and File Patterns

- Most Go packages expose a `main.go` file as the package entry surface, even when they are not executable packages.
- Templ output files follow the `*_templ.go` naming convention next to the `.templ` source.
- Migrations use timestamp-prefixed filenames such as `1697514430_create_postcards_table.go`.
- Handler packages are feature-oriented rather than HTTP-method oriented.

## Representative Paths

- `cmd/wga/main.go`
- `internal/handlers/landing/main.go`
- `internal/handlers/postcards/save.go`
- `internal/repositories/landing.go`
- `internal/crontab/postcard.go`
- `internal/assets/templ/pages/home.templ`
- `internal/assets/public/css/style.css`
- `playwright-tests/postcard.spec.ts`
