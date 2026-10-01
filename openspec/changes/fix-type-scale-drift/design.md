## Context

Tailwind 4 places utilities in `@layer utilities`. Unlayered rules beat every layer regardless of specificity, so the unlayered `.content` rule (`text-lg leading-normal`) overrode the `text-(length:--t-16) leading-[1.7]` utilities on each prose container. The unlayered `a:not([class])` default (semibold, underline) has the same effect on prose links.

## Decisions

**Move `.content` into `@layer components` and rewrite it in design tokens.** The rule's default becomes the design's running prose (`--t-16`, 1.7, `--wga-text`), and surfaces that choose another rung (Dual Mode and postcards at `--t-15`) keep it because utilities now win. Rewriting the rule in place keeps one owner for sanitised-HTML structure (lists, headings, code, quotations). The alternative, deleting `.content` and adding utilities to every container, would lose styling for headings and lists inside sanitised HTML, which utilities on the container cannot reach.

**Move `a:not([class])` into `@layer base`.** It is an element default; in the base layer the prose link rule in components can override its weight. Its declarations are unchanged, so classless links outside prose look the same.

**Focus ring uses `--wga-accent`.** Every per-palette focus override equalled that palette's `--wga-accent` exactly, and the design defines the ring in the accent role (P:508). Using the token in the base rule removes the duplicates and also corrects the default dark scheme without a palette attribute, which had no override.

**Rung mapping.** Each off-scale size takes the rung the design gives the same element: empty-state titles and confirmation marks `--t-18`, section and record titles `--t-20`, the guestbook empty state `--t-17`, tour blurbs `--t-18` (P:3030). The design has no standalone music player; its title takes `--t-22`, the rung the repository uses for comparable dialog and record headings.

## Risks / Trade-offs

- [Prose becomes smaller (18px to 17px) and lighter in colour] → this is the design's value; the browser assertion pins it at three widths in both schemes.
- [The synthetic fixture has no artwork commentary and no static-page link] → the browser assertion rewrites those two responses with representative sanitised prose inside the real container markup.
