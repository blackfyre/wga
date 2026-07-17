# Contributing

Thanks for contributing to the Web Gallery of Art.

## Development Setup

Use Mise as the primary development entrypoint:

```bash
mise install
mise run app:init-env
mise run dev
```

`mise run dev` starts the asset and template watchers, Mailpit (SMTP on port 1025 and HTTP API on port 8025), and MinIO; run the application separately with `mise run code:run`.

The main project tasks are:

- `mise run app:build` to install frontend dependencies, build assets, regenerate templates, and compile `dist/wga`
- `mise run app:run` to launch the built server from `dist/`
- `mise run code:run` to run the application directly with `go run ./cmd/wga --dev`

The server entrypoint is `cmd/wga/main.go`. Application code lives under `internal/`, frontend source files live under `resources/`, and browser tests live in `playwright-tests/`.

## Making Changes

Follow the current repo structure and workflow instead of older root-level path references. The main areas contributors touch are:

- `internal/handlers/`, `internal/utils/`, `internal/crontab/`, and `internal/migrations/` for Go application code
- `internal/assets/templ/` for Templ source files
- `resources/js` and `resources/css` for frontend source files
- `playwright-tests/` for browser coverage

Keep edits scoped to the task, preserve existing conventions, and update the relevant docs when command surfaces or repo structure change.

## Testing

Run the relevant automated checks before opening a PR:

```bash
go test ./... -cover
bunx playwright test
```

When working on frontend assets, use the existing watch scripts as needed:

```bash
bun run build:watch:css
bun run build:watch:js
```

Document any manual QA that matters for reviewers.

## Documentation Updates

Repository docs should follow code and configuration truth. If a command in docs conflicts with `mise.toml`, `package.json`, or the current code layout, update the docs as part of the same change.

Before editing repo docs, read [docs/documentation-maintenance.md](docs/documentation-maintenance.md). It lists the current source of truth files and the checklist for validating path, command, and environment wording.

Use the same canonical workflow terms across contributor docs: `mise run dev`, `mise run app:build`, `mise run app:run`, `mise run code:run`, `go test ./... -cover`, and `bunx playwright test`.

If docs conflict with code or config, the source of truth is the executable repo state described in that checklist.

When editing UI-related files, keep these boundaries clear:

- `.templ` source files live in `internal/assets/templ/`
- generated Go files live beside those sources as `*_templ.go`
- built frontend assets land in `internal/assets/public/`

Contributor-facing docs should use the current path model: `cmd/wga/main.go`, `internal/*`, `resources/*`, and `playwright-tests/*`.

## Security

Do not report vulnerabilities in public issues. Follow the reporting instructions in [SECURITY.md](SECURITY.md) and contact the maintainers privately at `wga+security@blackfyre.ninja`.
