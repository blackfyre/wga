## Why

The design audit found that rich-text prose ignores the design type scale. A legacy, unlayered `.content` rule sets 18px/27px in the ink role, overriding the `--t-16`/1.7 utilities every prose surface already declares, and renders prose links ink and semibold where the design uses the accent with a 1px underline. Elsewhere, two surfaces reference `--t-24`, which is not a rung of the 33-step scale, and 21 template uses of Tailwind's `text-lg`, `text-xl` and `text-2xl` bypass the scale. Per-palette focus-ring hex values duplicate the `--wga-accent` token, and `.home-prose` is no longer used.

## What Changes

- Running prose in sanitised rich text renders at the design's `--t-16` with 1.7 line-height in the text role, and a surface's own type-scale utilities take precedence over that default.
- Prose links render in the accent role with a 1px underline, no change of weight, and the secondary accent on hover.
- Headings, lists, code, quotations and strong text inside rich text keep their structure but use scale rungs and colour roles.
- Off-scale sizes (`--t-24`, `text-lg`, `text-xl`, `text-2xl`) map to the design's rung for each element.
- The focus ring takes `--wga-accent` directly, replacing the duplicated per-palette hex values; the unused `.home-prose` rule is removed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: adds the prose and link typography requirement under the existing release-reference type scale.

## Impact

- `resources/css/style.pcss` and the affected `.templ` sources only. No routes, data or behaviour other than presentation change.
- Non-goals: the page-title sizes on guided tours (`text-3xl` to `text-5xl`), work-grid breakpoints, Dual Mode's typeface, and tour-body sizing beyond the shared prose default; these belong to other audit changes.
