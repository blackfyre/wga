## Why

The Statistics art-form key draws two swatches per row. A legacy stylesheet rule adds a hard-coded swatch, in light-palette blues that ignore the active palette, theme and series ramp, beside the token swatch. The key therefore disagrees with the donut, most visibly in dark mode. The school charts also disagree with their key: the canvas paints "Other" as a solid tone, while the key shows the hatch the design requires, and each canvas draws a second legend of its own. Finally, the chart colours are not redrawn when only the palette changes. Separately, the page copy drifts from the design's Statistics screen (design-audit findings ST-1 and ST-2).

## What Changes

- Remove the legacy art-form swatch rule, so each key row draws one swatch.
- Make the server-rendered key the single source of each series colour. The charts take their colours from the key's swatches rather than from a second list in the script. "Other" draws as a hatch in both the key and the canvas, and the canvas no longer draws a duplicate legend.
- Redraw the charts when the palette changes as well as when the theme changes.
- Align the copy with the design. The artworks chart subtitle reads "STACKED BY SCHOOL · BIRTH PERIOD OF ARTIST". The table captions follow the design's "… — DATA" label style.
- **Deliberate deviations from the design:**
  - The design's footnote reads "RECOMPUTED {date}". The repo's "RECOMPUTED NIGHTLY" is not true either: the figures are recomputed on demand and cached for at most an hour. The footnote therefore reads "RECOMPUTED HOURLY". Showing a real recomputation date would need the handler to record when the cache was filled, which is outside this change.
  - The design shows no caption on the art-form table. The repo keeps it visible, because the existing contract requires a visible caption on every chart table.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `collection-discovery`: Statistics chart keys match their charts in every palette and theme.

## Impact

- `internal/assets/templ/pages/statistics.templ`, `resources/js/statistics.ts`, and the statistics rules in `resources/css/style.pcss`.
- The Statistics Playwright and templ tests.
- No route, data, handler or configuration changes.
