# Documentation Maintenance

Use this checklist whenever repository docs change. The goal is to keep contributor and maintainer guidance aligned to the files that actually define the current repo structure, commands, and environment model.

## Source of Truth

Check these files before editing docs that describe structure, commands, testing, or environment setup:

- `cmd/wga/main.go` for the server entrypoint
- `internal/*` for the current application package layout
- `mise.toml` for the pinned tools, canonical local scripts, and bundled services
- `package.json` for the active frontend build and watch scripts
- `.env.example` for documented environment variables such as `WGA_RECAPTCHA_SECRET` and `MAILPIT_URL`
- `playwright.config.ts` for Playwright runtime assumptions such as `baseURL` and the commented built-binary example
- `.github/workflows/pr-validation.yml` for PR-title validation
- `.github/workflows/playwright.yml` for backend CI quality gates, the Playwright flow, and the built-artifact execution model

## Documentation Change Checklist

- Confirm the current path model still starts at `cmd/wga/main.go`, `internal/*`, `resources/*`, and `playwright-tests/*`.
- Confirm built-binary instructions still use `dist/wga` rather than a root-level binary path.
- Confirm local mail wording still distinguishes the `mailhog` service from the `MAILPIT_URL` browser endpoint used by Playwright.
- Confirm the command being documented exists exactly where the docs claim it does, especially `mise install`, `mise run dev`, `mise run app:build`, `mise run app:run`, `mise run code:run`, `go test ./... -cover`, and `bunx playwright test`.
- Confirm environment-variable docs still match `.env.example`, including `WGA_RECAPTCHA_SECRET` and `MAILPIT_URL`.
- Confirm whether the docs change also requires updating secondary or generated guidance, such as `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, `.planning/codebase/*`, or historical summary docs.

## When Commands and Docs Conflict

If prose disagrees with code or config, the code and configuration files in the source-of-truth list win. Update the docs to match the executable repo state instead of documenting around the mismatch.

When a command is environment-sensitive, say so explicitly. Do not imply a workflow was executed locally if it was only verified from config or CI.

## Verification Commands

- `go test ./... -cover`
  Use this as the primary low-friction executable backend quality gate.
- `bunx playwright test`
  Use this for browser verification when the environment is ready with browser dependencies, a running app, and a reachable `MAILPIT_URL` endpoint.

## Scope Notes

- Keep primary contributor docs concise and consistent with each other.
- Update secondary or historical docs when they would otherwise preserve stale path or command wording as current truth.
- Treat CI workflow files as evidence for documented automation, not just background context.
