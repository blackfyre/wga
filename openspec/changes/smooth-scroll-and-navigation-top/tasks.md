## 1. Scroll behaviour

- [x] 1.1 Scroll in-page fragment jumps smoothly in the document and Dual Mode panes, only when no reduced-motion preference is set, keeping the existing reduced-motion override.
- [x] 1.2 Return to the top instantly after a main-area swap that pushes or replaces the URL, keep history restoration instant, and add unit tests for the decisions.

## 2. Verification

- [x] 2.1 Add a Playwright spec for smooth and reduced-motion anchors, top after navigation, unchanged fragment updates, Back restoration and fragment landing; run the Go, lint and browser suites.
