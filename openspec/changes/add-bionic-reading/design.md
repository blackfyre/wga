## Context

The shared public footer currently exposes cookie settings, while `resources/js/bootstrap.ts` initialises client-side preferences and shared HTMX behaviour. The visual-overhaul reference defines bionic reading as an optional DOM transformation: bold the opening part of each eligible word while retaining the same textual content.

The current application already uses local storage for its colour-theme preference and uses `htmx:load` to initialise behaviours after content is inserted. The change must work without changing server responses and must not add a non-essential cookie to the site's stated cookie model.

## Goals / Non-Goals

**Goals:**

- Give a visitor a reversible, locally persisted bionic-reading presentation preference.
- Limit transformation to running prose and preserve the original text when the preference is disabled.
- Apply the preference to content introduced through existing HTMX interactions.
- Provide an operable, accurately announced control only when JavaScript can perform the transformation.

**Non-Goals:**

- Claim a reading-speed or accessibility benefit.
- Persist the preference in a cookie, account, database, or server-side session.
- Transform navigation, figures, metadata, form controls, code, or existing semantic emphasis.
- Add a third-party dependency or modify the public API.

## Decisions

### Use a focused browser module, registered from the existing bootstrap entry point

The transformation and preference state will live in a dedicated TypeScript module, with bootstrap registering it once on initial page load. This keeps `bootstrap.ts` as the application composition point rather than extending its existing theme implementation with unrelated DOM-walking logic.

Alternative: add the logic directly to `bootstrap.ts`. Rejected because text transformation, reversal, and HTMX reprocessing form a separate behaviour that needs focused tests.

### Persist only in local storage

The module will store an explicit on/off value under a WGA-namespaced local-storage key and treat missing or inaccessible storage as off. It will update an HTML data attribute and all matching controls as its observable state.

Alternative: mirror the preference in a cookie for server-rendered initial state. Rejected because the transform is JavaScript-only, the existing theme preference does not require server state, and a cookie would expand the cookie/privacy surface without improving behaviour.

### Transform eligible text nodes in place and retain their source text

When enabled, a DOM tree walk will replace eligible prose text nodes with marked wrapper elements containing their original text in a data attribute and bold opening letter runs. Disabling will replace every marked wrapper with its stored source text. The walker will process only paragraphs and explicitly marked regions, skip nested marked output, `b`/`strong`, and `data-bionic="off"` subtrees.

Alternative: render alternate bionic HTML on the server. Rejected because it would require every prose source to supply a second representation and would make a local presentation preference part of response generation.

### Reprocess inserted HTMX content through the existing lifecycle

The module will apply the transform to an HTMX-loaded subtree whenever the stored preference is enabled. This follows the application's use of `htmx:load` for DOM behaviours and avoids rescanning all existing prose after each partial update.

Alternative: run the transform after every swap across `document.body`. Rejected because it does unnecessary work and risks reprocessing unrelated existing content.

### Progressively enhance a footer switch

The footer will contain a native button with switch semantics and an accessible name. It will be hidden until JavaScript initialises the feature, then its checked state will mirror the current preference. Visitors without JavaScript continue to receive unmodified server-rendered prose and no inert control.

## Risks / Trade-offs

- [DOM transformation could alter text selection or content replacement behaviour] → Store and restore each original text node exactly; restrict processing to prose; cover enable/disable restoration and HTMX insertion in tests.
- [Existing semantic emphasis could be visually flattened] → Exclude text inside `b` and `strong`, and support `data-bionic="off"` for explicit exclusion.
- [A full document scan could be costly on repeated HTMX interactions] → Process the initial body once and subsequent HTMX targets only.
- [Visitors could mistake the control for a proven accessibility treatment] → Keep the neutral “Bionic reading” label and make no performance or accessibility claim in UI or documentation.

## Migration Plan

No data migration is required. Deployment adds a client-side preference that defaults to off. Removing the feature later only leaves an unused local-storage key; public content remains unaffected.

## Open Questions

None.
