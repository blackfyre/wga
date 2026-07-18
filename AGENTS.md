# WGA Agent Guide

## Application boundaries

- `cmd/wga/main.go` creates the PocketBase app, then registers handlers, hooks, cron jobs, and migrations before `app.Start()`.
- Route modules are registered from `internal/handlers/main.go`; add a new handler package there rather than looking for a central route table.
- Add PocketBase migrations as timestamped files in `internal/migrations/` that call `m.Register` from `init()`. The entrypoint blank-imports this package and disables automigration; run the built binary's `migrate` command explicitly when needed.
- Edit Templ sources in `internal/assets/templ/`, then run `templ generate`. Adjacent `*_templ.go` files are generated and Git-ignored: do not edit or commit them.
- Edit frontend sources in `resources/js/` and `resources/css/`; `bun run build` writes generated JS/CSS to `internal/assets/public/{js,css}`, which the Go binary embeds. `internal/assets/views/` and `internal/assets/reference/` are also embedded at build time.
- The active Tailwind 4/daisyUI theme is in `resources/css/style.pcss`; UI work must also follow `.github/instructions/daisyui.instructions.md`.

## Environment and development

- Use Go 1.25.2 (`go.mod`/`mise.toml`), Bun, and Templ. `devenv shell` is the documented development environment; `mise` pins the same toolchain and exposes equivalent tasks as `mise run <task>`.
- Create `.env` from `.env.example` (`mise run app:init-env`). `godotenv.Load()` reads the default `.env` from the process working directory: `code:run` uses the repository root, while `app:run` changes into `dist/`.
- `wga_data` is likewise relative to the process working directory. `app:run` uses `dist/wga_data`; clear the data directory used by the launcher rather than assuming root `wga_data` is the active one.
- `devenv up` starts JS/CSS/Templ watchers, Mailpit, and MinIO, but not the application server. Start it separately with `code:run`, or use `app:build` followed by `app:run`.
- `app:build` runs `bun install`, `bun run build`, `templ generate`, `go mod tidy`, then builds `dist/wga`. `seed:images` is registered only when `WGA_ENV=development`.

## Verification and workflow

- Backend CI order is `go mod tidy`, `go vet ./...`, then `go test ./... -cover`. For a focused check, use commands such as `go test ./internal/handlers/dual -run '^TestResolvePaneTarget$'`.
- `mise run check` runs the local Go pre-commit checks (`go vet` and `golangci-lint`), not the test suite. `.pre-commit-config.yaml` is generated; do not edit it.
- Playwright has no active `webServer` setting. Before `bunx playwright test` (or one spec such as `bunx playwright test playwright-tests/artwork-search.spec.ts`), start the app and set `WGA_PROTOCOL`, `WGA_HOSTNAME`, and a reachable `MAILPIT_URL`; the postcard spec queries the Mailpit API.
- The full Go suite includes a mail-send test that skips only when no `sendmail` executable is available.
- `biome.json` configures JS/TS tabs, double quotes, and import organisation. The Playwright CI workflow also runs Prettier on changed JS and Markdown files.
- PR titles must use one of the Conventional Commit types enforced by `.github/workflows/pr-validation.yml`: `feat`, `fix`, `docs`, `test`, `ci`, `refactor`, `perf`, `chore`, `revert`, or `build`.
- Non-`main` deployment runs only when the head commit message contains `deploy-dev`; release tags matching `v*.*.*` invoke GoReleaser.
- When changing repository documentation, read `docs/documentation-maintenance.md`; it identifies the authoritative config and CI sources, including the Mailpit service and `MAILPIT_URL` endpoint.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **wga** (4755 symbols, 13596 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({search_query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/wga/context` | Codebase overview, check index freshness |
| `gitnexus://repo/wga/clusters` | All functional areas |
| `gitnexus://repo/wga/processes` | All execution flows |
| `gitnexus://repo/wga/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
