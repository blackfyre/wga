## Context

The art-form key and the school key are server-rendered swatches that use the `--wga-series-*` tokens. The Chart.js script kept its own token lists for the same series. The two lists matched for the seven tones but not for "Other" (a hatch in the key, and a solid `--wga-fill-line` in the canvas). Nothing kept them aligned beyond a code comment.

## Decisions

### The key is the colour source; the script reads it

Each key swatch carries the token it paints (`data-series-token`), plus `data-series-fill="hatch"` for "Other". The script resolves each dataset's colour from the swatches of that chart's key: for the art-form donut, the swatches in row order; for the school charts, the school legend in the same section. It builds a canvas hatch pattern for hatched series. The ramp therefore lives in exactly one place, the templ, and the canvas cannot drift from the key it is read against.

The alternative was to keep the token lists in TypeScript and have the server mirror them. That keeps two lists and only moves the drift elsewhere.

### The canvas draws no legend

The server-rendered school legend is the key with or without JavaScript. The duplicate canvas legend is switched off, so each chart has one key.

### Observable colour contract for tests

After drawing, each canvas exposes the colours it was given as `data-series-colours` (a JSON array; a hatch is recorded as `hatch:<colour>`). This follows the existing `data-chart-animation` test hook. It lets a browser test compare each key swatch's computed colour with the chart's series colour without sampling canvas pixels.

## Risks

- A key with no swatches would leave the chart without colours. The script falls back to the series ramp by index, so the chart still renders.
