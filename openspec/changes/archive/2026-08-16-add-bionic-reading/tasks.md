## 1. Bionic-reading behaviour

- [x] 1.1 Create a focused browser module that reads and writes the off-by-default local-storage preference, exposes the current state to the document, and updates matching controls.
- [x] 1.2 Implement reversible text-node transformation for paragraphs and explicitly marked regions, preserving source text and excluding marked output, existing `b`/`strong` content, and `data-bionic="off"` subtrees.
- [x] 1.3 Add focused tests covering default state, persisted state, scoped transformation, excluded content, and exact restoration after disabling.

## 2. Shared-page integration

- [x] 2.1 Register the bionic-reading module from the existing browser bootstrap entry point and apply an enabled preference to HTMX-loaded content without reprocessing existing transformed prose.
- [x] 2.2 Add an accessible, progressively enhanced Bionic reading switch to the shared footer; ensure its visual and announced state follows the preference.
- [x] 2.3 Add integration coverage for footer activation and eligible prose introduced by an HTMX update.

## 3. Verification

- [x] 3.1 Run the focused browser tests and confirm enabled, disabled, restored, and HTMX-inserted prose satisfies the bionic-reading specification.
- [x] 3.2 Run the frontend build and confirm generated assets compile successfully.
