# Concerns

## Architectural Coupling

- The app is heavily coupled to PocketBase for routing, persistence, settings, cron, mail, and file handling.
- That keeps the project compact, but it also means upgrades or platform changes can have broad blast radius.

## Documentation Drift

- Phase 2 corrected the contributor-facing docs so they now point at `cmd/wga/main.go`, `internal/*`, and the current development commands.
- Documentation drift is still a regression risk if future edits stop checking `cmd/wga/main.go`, `internal/*`, `devenv.nix`, `package.json`, and `playwright.config.ts` before changing repo guidance.
- The maintenance risk is no longer stale contributor docs today; it is reintroducing stale path or command terminology later.

## Generated and Built Files in Repo

- Generated Templ files and compiled frontend outputs are committed.
- This reduces setup friction but increases churn and review noise.
- Contributors have to remember to regenerate assets/templates when changing source files.

## Testing Depth

- Core helper logic has decent unit coverage, but route-level and end-to-end coverage is still selective.
- External integration paths such as email delivery, contributor API behavior, and PocketBase route wiring have more operational risk than current tests appear to cover.

## Mixed Query Styles

- Some data access uses PocketBase record helpers while other paths use raw SQL through `app.DB().NewQuery(...)`.
- That is pragmatic, but it creates more than one persistence idiom to maintain.

## Environment Sensitivity

- Critical runtime behavior depends on environment variables for app URL, SMTP, S3, admin bootstrap, and CAPTCHA.
- Misconfiguration at migration time can silently produce bad app settings.
- Local dev defaults in `devenv.nix` help, but production parity still depends on careful env management.

## Potential Security and Privacy Watchpoints

- Admin credentials are seeded from env vars in migrations.
- File-download hooks currently log request events; logging should avoid leaking sensitive record/file details in production.
- Generated codebase documentation must never include actual secret values from local `.env` files.

## Code Quality Hotspots

- Some packages still carry older comment style and broad files, especially in utility and cron areas.
- There is at least one oddly named file, `internal/handlers/musics._go`, which looks accidental or dead.
- TODO comments remain in active request paths such as the landing page metadata handling.

## Operational Fragility

- Cron postcard sending performs send and record-update work inline; partial failure handling looks basic.
- Contributor API requests add an external dependency path that can fail or slow requests.
- Playwright postcard tests depend on a mail-capture service URL, which adds environment setup sensitivity for CI/local runs.

## Recommended Attention Areas

- Guard against reintroducing stale path or command terminology.
- Expand request/route tests around critical forms and cron side effects.
- Review dead or oddly named files.
- Audit platform-specific coupling before any major PocketBase upgrade.
