## 1. Shared keyboard contracts

- [x] 1.1 Replace the browser route map with a server-rendered screen registry that supplies supported section letters, numbers, URLs, help rows, and palette rows.
- [x] 1.2 Add the hint bar, complete shortcut help, desktop search, and mobile navigation contracts required by the shared layer.

## 2. Global interaction behaviour

- [x] 2.1 Implement section-letter and two-digit section jumps with a one-second numeric buffer, while preserving reserved movement keys.
- [x] 2.2 Implement responsive `/`, Ctrl/Cmd+K, `?`, and Escape behaviour, including field blur, mobile-menu dismissal, and editable-control exclusions.

## 3. Caret traversal

- [x] 3.1 Mark supported artist, artwork grid/list, and guestbook lists with visible caret markup and declared column counts.
- [x] 3.2 Implement clamped row-aware up/down and J/K traversal, adjacent left/right traversal, Enter activation, and HTMX caret reset.

## 4. Palette suggestions and resilience

- [x] 4.1 Render and locally filter palette sections; debounce record suggestions for 140 ms and send the remaining result capacity.
- [x] 4.2 Enforce endpoint query, capacity, ordering, public-record, rate-limit, and expired-client cleanup contracts.

## 5. Verification

- [x] 5.1 Add focused Go tests for bounded suggestion requests and limiter expiry cleanup.
- [x] 5.2 Add Playwright coverage for section and numeric jumps, desktop/mobile `/`, palette selection, editable controls and Escape, caret movement boundaries, and HTMX replacement.
