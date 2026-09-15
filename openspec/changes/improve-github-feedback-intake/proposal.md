## Why

The pre-release feedback link currently opens the repository's issue list, while its generic Markdown templates do not reflect the application's established feedback categories. GitHub should provide a clear interim intake without dismantling the private in-app workflow intended for the first production release.

## What Changes

- Route the pre-release feedback entry point to GitHub's issue template chooser.
- Replace the generic Markdown templates with structured public issue forms aligned to general feedback, collection corrections, technical problems, and suggestions.
- Make the public nature of GitHub submissions explicit and avoid requesting private contact details.
- Preserve the dormant in-app feedback handler, form, persistence, and category model for restoration at the first production release.
- Do not automate the future release switch or remove existing private feedback infrastructure.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-feedback`: Define the interim public GitHub intake and preservation of the in-app production feedback path.

## Impact

- `.github/ISSUE_TEMPLATE/` issue forms and chooser configuration.
- The shared Templ layout feedback destination and its focused Go/browser coverage.
- Existing `/feedback` handlers, components, migration, and persistence remain unchanged.
