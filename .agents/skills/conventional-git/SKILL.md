---
name: conventional-git
description: "Creates Conventional Commit messages and matching pull-request titles and branch names. Use when preparing a commit, pull request, or branch."
---

# Conventional Git Naming

Use this skill when proposing or creating a branch, commit, or pull request.

## Guardrails

- Derive the name from the actual, intended change. Inspect `git status`, the relevant diff, and recent log messages before naming it.
- Do not create, commit, push, or switch branches unless the user explicitly requests that operation.
- Keep one commit focused on one coherent change. Split unrelated changes instead of hiding them behind a broad subject.
- Never include credentials, tokens, private URLs, or sensitive personal data in a branch name, commit subject, or pull-request title.

## Type

Choose the narrowest supported Conventional Commit type:

| Type | Use for |
| --- | --- |
| `feat` | New user-facing capability |
| `fix` | Correcting faulty behaviour |
| `docs` | Documentation only |
| `test` | Tests only |
| `ci` | Continuous-integration configuration |
| `refactor` | Internal restructuring with unchanged behaviour |
| `perf` | Measurable performance improvement |
| `chore` | Maintenance not covered above |
| `revert` | Reverting an earlier change |
| `build` | Build system or dependency tooling |

This repository validates pull-request types against exactly this list.

## Commit and Pull-Request Title

Use the same conventional subject for the commit and PR title unless the PR deliberately groups several independently named commits:

```
<type>(<optional-scope>): <imperative, lower-case summary>
```

- Omit the scope unless it makes the affected area materially clearer.
- Start the summary with a verb, use lower case, avoid a trailing full stop, and keep it concise.
- State the outcome, not the implementation detail or ticket number.
- Use `!` after the type or scope only for a deliberate breaking change; explain it in the commit body or PR description.

Examples:

```
feat(tours): add guided tour landing page
fix(postcard): preserve delivery retry state
docs: clarify local development setup
build: update frontend asset pipeline
```

For an issue-backed change, put the issue reference in the PR description or commit body unless the repository's established local convention requires it in the subject.

## Branch Name

Use a lower-case, slash-separated name:

```
<type>/<short-kebab-case-summary>
```

Optionally prefix the summary with an issue identifier when one is authoritative:

```
<type>/<issue-id>-<short-kebab-case-summary>
```

Examples:

```
feat/guided-tour-landing-page
fix/197-postcard-retry-state
docs/local-development-setup
```

- Use the same type as the eventual commit and PR where practical.
- Keep the summary brief, specific, and free of dates, usernames, spaces, underscores, and punctuation.
- Do not use `main`, `master`, release names, or a branch name that suggests completed work for an unfinished change.

## Before Reporting or Creating

1. Confirm the type matches the diff and repository conventions.
2. Check that the subject and branch summary describe the same outcome.
3. Confirm the PR title begins with one of the repository-approved types.
4. When the user asks only for suggestions, provide a proposed branch, commit, and PR title without running Git write operations.
