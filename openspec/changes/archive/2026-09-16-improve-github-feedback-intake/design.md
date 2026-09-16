## Context

The shared layout currently exposes an ordinary link to the repository's open-issue listing. GitHub has two generic Markdown templates, while the retained HTMX feedback workflow already defines the durable category model: general, correction, technical, and suggestion. The private workflow must remain available for restoration at the first production release.

## Goals / Non-Goals

**Goals:**

- Give pre-release visitors a direct, structured route for filing feedback.
- Keep GitHub and in-app feedback categories conceptually aligned.
- Preserve the complete private feedback implementation for the later release switch.

**Non-Goals:**

- Removing, refactoring, or changing the `/feedback` workflow or persistence schema.
- Automatically selecting the intake channel from the runtime environment or version.
- Implementing the first-production-release switch back to the in-app dialog.

## Decisions

### Use the GitHub issue chooser as the interim entry point

The layout link will target `/issues/new/choose`, allowing visitors to select the report type before entering details. Linking directly to one generic issue form would reduce the value of category-specific prompts and labels.

### Use four YAML issue forms aligned with the application categories

Separate forms allow each category to ask only for actionable evidence and apply its existing repository label where applicable. Chooser configuration will disable blank public issues so visitors consistently receive privacy guidance and structured prompts.

### Keep the private intake untouched

The handler, Templ dialog, browser behaviour, migration, and feedback collection will not be removed or repurposed. Restoring the button's HTMX contract will be an explicit first-production-release task, avoiding a hidden environment-dependent switch before that release is ready.

## Risks / Trade-offs

- **GitHub requires an account and submissions are public** -> Every form carries prominent privacy guidance; the private in-app intake remains the intended production destination.
- **Two intake representations can drift** -> Use the same four category meanings and verify their presence in repository tests.
- **The retained `/feedback` path remains undiscoverable temporarily** -> Document this as deliberate and leave its existing focused tests intact.
