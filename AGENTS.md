# Repository Guidelines

## Project Structure & Module Organization

The Go entrypoint `main.go` wires PocketBase integration and Echo handlers in `handlers/`, with shared helpers in `utils/` and scheduled jobs in `crontab/`. UI templates originate in `assets/templ/` (components, layouts, pages) and render into Go template fragments in `assets/views/`, while generated static assets land in `assets/public/`. Front-end source lives in `resources/js` and `resources/css`, database migrations in `migrations/`, and browser specs in `playwright-tests/`.

## Build, Test, and Development Commands

Use `devenv` as the primary CLI entrypoint for general project tasks. Run `devenv shell` to enter the development environment, then invoke the scripts defined in `devenv.nix` directly.

- `devenv up` starts the local development processes and bundled services such as template watching, asset watchers, MailHog, and MinIO.
- `app:init-devenv` bootstraps `devenv.local.nix` from the local stub for first-time setup.
- `app:generate-templates` regenerates Go templates from `.templ` sources.
- `app:tidy` refreshes generated templates and runs `go mod tidy`.
- `app:build` installs front-end dependencies, builds assets, refreshes generated code, and compiles `dist/wga` from `./cmd/wga`.
- `app:run` launches the built server from `dist/` in development mode, while `code:run` runs the app directly with `go run ./cmd/wga --dev`.
- `app:reboot` rebuilds the app, clears local `wga_data`, and restarts the development server.
- `go test ./... -cover` runs unit tests and reports package coverage.
- `bunx playwright test` executes the end-to-end suite in `playwright-tests/`.

Use raw `go`, `bun`, and `templ` commands for focused work when needed, but prefer the repo-defined `devenv` scripts for common workflows so contributors share the same toolchain and sequencing.

## Coding Style & Naming Conventions

Format Go code with `go fmt ./...` and keep package names lower case, mirroring their folder (for example `handlers/dual`). Follow Biome's defaults (`biome.json`): tab indentation, double quotes, sorted imports, and run `bunx @biomejs/biome format .` before committing front-end changes. Name Templ views with kebab-case filenames that match their route fragment and avoid committing generated build artefacts outside `assets/public/`.

## Testing Guidelines

Co-locate Go tests with their source using the `_test.go` suffix and table-driven cases for branching logic. Expand Playwright coverage when UI flows change; prefer data-attribute selectors and use `bunx playwright test --headed` for debugging. Document any manual QA in the PR description and keep `go test` coverage from regressing when adding new features.

## Commit & Pull Request Guidelines

Write imperative, 50–72 character commit subjects; conventional prefixes (`fix:`, `feat:`, `chore:`) keep history searchable as seen in recent commits. Clean up WIP commits via rebase before opening a PR. PRs should outline motivation, list major changes, link GitHub issues, and attach screenshots or recordings for UI-impacting work. Note which automated tests ran and call out any follow-up tasks.

## Security & Configuration Tips

Store secrets in a local `.env`; never commit real credentials. Use `devenv up` to provision local services such as MinIO and MailHog, and align hostnames with `WGA_HOSTNAME` defaults. Review `SECURITY.md` before reporting vulnerabilities and coordinate fixes privately when sensitive data is involved.
