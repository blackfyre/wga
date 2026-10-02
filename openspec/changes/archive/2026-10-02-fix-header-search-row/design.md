## Context

The desktop header (`TopNav()` in `internal/assets/templ/components/nav.templ`) is a logo row and a navigation row. On the logo row the search form was `w-[190px]` below 1080px and `w-[340px]` from 1080px, and its label, SEARCH and `CTRL K` were all shrinkable flex items, so the 190px was shared between them. The shell's `.container` rule is unlayered and so overrides the header's `px-0` utilities: content is the viewport minus 56px between 720 and 1079px and minus 80px from 1080px.

Measured with the system monospace face (Noto Sans Mono locally, DejaVu Sans Mono on CI; both 0.6em advances), the logo block is 370px wide, governed by the strapline. SEARCH is 53px and `CTRL K` 65px at their natural widths. The seven navigation links are 575px together and MORE is 52px.

## Decisions

**Natural-width controls and a flexible field.** SEARCH and `CTRL K` are `shrink-0 whitespace-nowrap`; the label is `min-w-0 flex-1` and the form is `w-full max-w-[340px]` inside a `min-w-0 flex-1` wrapper aligned to the end of the row. From 1080px the form still reaches its 340px maximum, so the wide layout is unchanged (input 190px). The logo is `shrink-0`, so the brand never wraps to make room for search.

**A 140px field minimum.** At 140px the 14px placeholder shows "Search artist, title," and a typed query of about 18 characters, which is enough to read and edit a name; below that the field reads as a stub. The minimum is met by budgeting the row's gaps (below), not by a hard `min-width`: a platform with a classic, non-overlay scrollbar narrows the page by about 17px while media queries still see the full viewport, and a hard minimum would then push the row into horizontal overflow at 720px. With such a scrollbar the field shrinks to about 127px at 720px instead, and reaches 140px again by about 735px.

**Gaps that make 720px fit.** At 720px the logo row has 664px. The design's 16px row gap (replacing the repository's 24px) plus SEARCH, `CTRL K` and the 16px form gaps leaves 136px of field, short of the minimum. Below 1080px the form gap becomes 8px, leaving 144px. From 1080px the form keeps its 16px gaps. The design's exact search-group gap was not readable, so this is the smallest tightening that meets the minimum.

**Navigation row at 834px.** The row has 778px at 834px. With the 26px item gap used from 1080px and 24px before MORE it needs 803px, so INSPIRATION wrapped. Below 1080px the item gap and the gap before MORE become 18px, needing 753px. That fits at 834px and above even when a 17px classic scrollbar leaves 761px. At 720px the row has 664px and cannot fit even with 6px gaps, so INSPIRATION wraps onto a second line while MORE stays at the end of the first, keeping every destination visible. Moving destinations into MORE at that width would change the information architecture and is out of scope.

**Wordmark and strapline.** The desktop wordmark takes the design's `--t-14`, weight 600, 3px tracking and 1.15 line height; the strapline takes `--t-10`, 1px tracking, the `--wga-faint` role and a 4px top margin. The repository's `--t-*` tokens were matched to the design's raised scale, so the same token names give the design's values (15.5px and 12px). The mobile header keeps its current type because the larger wordmark would wrap beside the menu button at 375px; only the strapline text, which both branches share, changes to the design's lowercase ordinals.

**MORE marker.** The summary uses `list-none` and hides `::-webkit-details-marker`, then renders `▾` in an `aria-hidden` span, so its accessible name stays "MORE".

## Risks / Trade-offs

- [Font metrics differ on a visitor's system] → monospace faces share 0.6em advances, the field has 4px of slack at 720px, and the field shrinks rather than overflowing if a face is wider.
- [A classic scrollbar narrows the page] → headless Chromium hides scrollbars, so the browser check simulates a 17px scrollbar by narrowing the body at 720px and 834px and asserts no row overflow, a one-line cue, a field of at least 120px and, at 834px, a one-line navigation row.
- [The navigation row still wraps at 720px] → the browser check asserts a single line only from 834px and asserts no horizontal overflow at 720px.
- [Mobile and desktop wordmarks now differ in type] → the mobile layout is constrained by the menu button; aligning it is left to a mobile-specific change.
