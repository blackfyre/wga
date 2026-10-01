## Context

See proposal.md. The design is read through the `DesignSync` tool (`get_project`, `list_files`, `get_file`). It is read-only, available only in the primary session after `/design-login`, and caps each file at 256 KiB. `WGA Prototype.dc.html` is about 709 KiB, so it cannot be read whole through the tool.

## Decisions

**Track implementation state in a repository document.** `docs/design-sync.md` records the last implemented changelog entry by date and title, plus any entry being implemented in an open change. Each change that implements an entry updates it in the same pull request. A marker in the design project was rejected because the design is read-only from this repository and the record must travel with the code that implements it.

**Turn changelog entries into OpenSpec changes, not direct edits.** The `design-intake` skill lists entries newer than the recorded one, maps each entry's downstream notes onto real repository files, and proposes one OpenSpec change per coherent entry without implementing it. Implementation then follows the normal OpenSpec workflow.

**Check parity against a confirmed local export.** Because the prototype exceeds the `get_file` cap, parity checks use the changelog plus a local export at `../wga-visual-overhaul/project/`, which must be refreshed or confirmed current against the live changelog before use. Keeping an embedded copy was rejected: it drifts silently and enlarges every build.

**Re-point the footer-replacement test at a real page.** The fixture route existed only to swap a server-rendered footer with HTMX. On a public page, the test retargets an existing processed `hx-get` navigation link at `footer`, so the real page's server-rendered footer is swapped through the same HTMX lifecycle without a development-only route.

## Risks / Trade-offs

- [The local export can be stale] → the skill and the tracking document require confirming it against the live changelog before a parity check.
- [The tracking document can fall behind] → the skill requires updating it in the implementing pull request, and an in-progress entry is recorded so whichever change merges second reconciles it.
