# WGA Agent Guide

## Application boundaries

- `cmd/wga/main.go` creates the PocketBase app, then registers handlers, hooks, cron jobs, and migrations before `app.Start()`.
- Route modules are registered from `internal/handlers/main.go`; add a new handler package there rather than looking for a central route table.
- Add PocketBase migrations as timestamped files in `internal/migrations/` that call `m.Register` from `init()`. The entrypoint blank-imports this package and disables migration-file generation; `serve` applies pending migrations before listening, while the built binary's `migrate` command remains available for explicit operations.
- Edit Templ sources in `internal/assets/templ/`, then run `templ generate`. Adjacent `*_templ.go` files are generated and Git-ignored: do not edit or commit them.
- Edit frontend sources in `resources/js/` and `resources/css/`; `bun run build` writes generated JS/CSS to `internal/assets/public/{js,css}`, which the Go binary embeds. `internal/assets/views/` and `internal/assets/reference/` are also embedded at build time.
- The active Tailwind 4/daisyUI theme is in `resources/css/style.pcss`; UI work must also follow `.github/instructions/daisyui.instructions.md`.
- Treat `.templ` files as authoritative frontend source. Treat HTMX markup, its Go handler, rendered fragment, and swap target as one interaction contract; prefer server-rendered HTML and minimal JavaScript.
- Keep WGA as one deployable application. Extend the owning feature package and use explicit contracts across capability boundaries rather than reaching into another feature's persistence helpers.
- Treat handlers and hooks as framework adapters: parse input, obtain request context, invoke the owning workflow, and map the result. Keep non-trivial business rules, state changes, and external side-effect ordering outside request handlers.
- Load deployment settings only through `internal/config`; feature code must not read environment variables or `.env` files directly. Add parsing, validation, and focused tests there for new settings.
- Use `logging.RequestLogger` for request-scoped work, `logging.RunLogger` for cron or background work, and `logging.Redact` for sensitive values. Request IDs are server-generated and public request headers are not trusted as their source.
- For new direct third-party integrations, persist work before an external side effect when recovery or retry matters, and define idempotency before retries can repeat it.

## Documentation map

- `docs/development-guide.md` contains durable application design, configuration, external-work, logging, privacy, and delivery guidance.
- `docs/documentation-maintenance.md` identifies the sources of truth and checks required when repository documentation changes.
- `docs/features/` contains feature specifications and acceptance criteria.
- Issue plans, review notes, and historical summaries are task-specific context, not current implementation guidance unless explicitly stated otherwise.

## Environment and development

- Use Go 1.26.5 (`go.mod`/`mise.toml`), Bun, and Templ. `devenv shell` is the documented development environment; `mise` pins the same toolchain and exposes equivalent tasks as `mise run <task>`.
- Create `.env` from `.env.example` (`mise run app:init-env`). `godotenv.Load()` reads the default `.env` from the process working directory: `code:run` uses the repository root, while `app:run` changes into `dist/`.
- `wga_data` is likewise relative to the process working directory. `app:run` uses `dist/wga_data`; clear the data directory used by the launcher rather than assuming root `wga_data` is the active one.
- `mise run dev` brings up the Podman Compose Mailpit and Garage services, then starts JS/CSS/Templ watchers, but not the application server. Start it separately with `code:run`, or use `app:build` followed by `app:run`.
- `app:build` runs `bun install`, `bun run build`, `templ generate`, `go mod tidy`, then builds `dist/wga`. The embedded synthetic-data migration initialises a fresh data directory on first server start if not configured otherwise. `seed:images` is registered only when `WGA_ENV=development`.

## Verification and workflow

- Backend CI order is `go mod tidy`, `go vet ./...`, then `go test ./... -cover`. For a focused check, use commands such as `go test ./internal/handlers/dual -run '^TestResolvePaneTarget$'`.
- `mise run check` runs the local Go pre-commit checks (`go vet` and `golangci-lint`), not the test suite. `.pre-commit-config.yaml` is generated; do not edit it.
- Playwright has no active `webServer` setting. Before `bunx playwright test` (or one spec such as `bunx playwright test playwright-tests/artwork-search.spec.ts`), start the app and set `WGA_PROTOCOL`, `WGA_HOSTNAME`, and a reachable `MAILPIT_URL`; the postcard spec queries the Mailpit API.
- The full Go suite includes a mail-send test that skips only when no `sendmail` executable is available.
- `biome.json` configures JS/TS tabs, double quotes, and import organisation. The Playwright CI workflow also runs Prettier on changed JS and Markdown files.
- PR titles must use one of the Conventional Commit types enforced by `.github/workflows/pr-validation.yml`: `feat`, `fix`, `docs`, `test`, `ci`, `refactor`, `perf`, `chore`, `revert`, or `build`.
- Non-`main` deployment runs only when the head commit message contains `deploy-dev`; release tags matching `v*.*.*` invoke GoReleaser.
- When changing repository documentation, read `docs/documentation-maintenance.md` and `docs/development-guide.md`; the maintenance guide identifies the authoritative config and CI sources, including the Mailpit service and `MAILPIT_URL` endpoint.

## Model-family assurance

- For material or high-risk changes, use different model families for substantive implementation and final assurance where practical: DeepSeek implementation uses OpenAI review or verification; OpenAI implementation uses DeepSeek review or verification.
- Apply this especially to authentication, authorisation, tenancy/isolation, security boundaries, database migrations, financial behaviour, concurrency, infrastructure, public API changes, difficult debugging, and large cross-component changes.
- Do not require cross-family assurance for trivial or deterministic work. Agents from the same model family are not independent evidence merely because they have different role prompts.
- Objective verification remains stronger evidence than either model family: runtime behaviour; compiler/type checker; database constraints; static analysis; automated tests; source inspection; then model judgement.

<!-- workshop:start -->
## Workshop workflow

### Source of truth

- Treat OpenSpec specs and active change artifacts as authoritative for intended behavior and scope.
- Do not invent requirements to make an implementation feel more complete.
- If code and the spec disagree, surface the conflict rather than silently choosing whichever is convenient.
- Read only the OpenSpec artifacts and repository files required for the current task. Do not preload the whole project or every change artifact.

### Ownership

- One primary implementation agent owns an atomic task from understanding through code, tests, deterministic verification, and correction.
- Work on one OpenSpec task at a time unless the user explicitly asks for a batch.
- Completing one task does not authorize continuing into the next task.

### Complexity routing

Classify the current task internally before implementation. Do not produce a ceremony-filled classification report.

- Tier 0: trivial/local/mechanical. No subagent.
- Tier 1: normal implementation. Implement directly; use `explore` only if repository reconnaissance would materially reduce uncertainty; run deterministic checks; use `review` after checks pass.
- Tier 2: ambiguous, cross-cutting, difficult debugging, or significant architectural tradeoff. Use `architect` once before implementation; then implement, verify, and use `review`.
- Tier 3: authentication, authorization, payments, secrets, destructive migration, data-loss risk, or another critical boundary. Use either `architect` or `security` before implementation, whichever addresses the dominant risk; then implement, verify, and use `review`.

Hard budget: at most two subagent calls for one task, including the final review. Never call a specialist merely because its name matches a file type or technology.

### Delegation

- Subagents are consultations, not managers and not workers that own slices of the implementation.
- Subagents must never delegate further.
- Do not ask multiple agents the same question to manufacture consensus.
- Do not hand a subagent a transcript or broad project dump. Provide only the task/question, relevant spec requirement, relevant paths, and the minimal diff/evidence it needs.
- Prefer `explore` for locating facts. Prefer direct repository tools when the answer is already obvious from known paths.

### Server-rendered frontend boundary

- In Go `templ` + HTMX projects, frontend behavior commonly spans the handler, `.templ` component, returned fragment, swap target, CSS, and small vanilla-JS behavior. Treat that as one coherent implementation boundary owned by the primary agent.
- Do not create or summon a frontend subagent merely because `.templ`, HTML, CSS, or JavaScript is involved.
- For user-facing UI work, load `frontend-design` when visual/interaction/accessibility behavior changes and `htmx` when HTMX request/fragment/swap behavior changes. Skill loading is on-demand context and does not count against the two-subagent budget.
- Prefer reusable templ fragment components that can be composed into full pages rather than duplicated full-page and HTMX markup.
- Treat the HTMX interaction as an HTTP/rendering contract: initiating element -> request -> Go handler -> templ component -> response -> target/swap -> resulting DOM/state.
- Vanilla JavaScript attached to swappable markup must survive repeated HTMX swaps; prefer native HTML, HTMX behavior, delegated listeners, or idempotent lifecycle initialization.
- When UI behavior changes, independent review should load `web-design-guidelines`; if the HTMX contract changed, it may also load `htmx`.
- Compilation proves compilation, not browser interaction. Do not claim visual, focus, swap, or interaction behavior was verified unless it was actually exercised by an appropriate deterministic/browser check.

### Implementation loop

1. Identify the single atomic task and its acceptance criteria.
2. Inspect only relevant code/specs.
3. If complexity warrants it, spend at most one pre-implementation specialist call.
4. Implement the task end-to-end in the primary session.
5. Run the project's deterministic verification in the cheapest useful order: formatter -> compile/typecheck -> focused tests -> broader tests -> lint/static analysis, adjusted to the repository's actual tooling.
6. Correct deterministic failures in the primary session.
7. After deterministic checks pass, request one independent `review` when the tier requires it.
8. Apply valid review findings and rerun the affected deterministic checks.
9. Stop after two remediation cycles. Record the unresolved failure and evidence instead of starting an agent recursion.
10. Mark an OpenSpec task complete only after its acceptance criteria and required verification are satisfied.

### Output discipline

- Prefer patches, test results, concise decisions, and structured findings over narrative reports.
- Do not narrate routine repository exploration.
- Do not restate the full spec or task before acting.
- Never claim a check passed unless it was actually run or the user explicitly accepted a non-executed verification plan.
- Final task reports should normally contain: what changed, verification performed, review outcome, and any unresolved issue.
<!-- workshop:end -->
