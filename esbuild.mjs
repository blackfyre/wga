import * as esbuild from "esbuild";
import { sassPlugin } from "esbuild-sass-plugin";
import postcss from "postcss";
import autoprefixer from "autoprefixer";
import purgeCSSPlugin from "@fullhuman/postcss-purgecss";
import { copy } from "esbuild-plugin-copy";
import fs from "node:fs";

console.info("🚀 Starting build 🚀");

let result = await esbuild.build({
  entryPoints: ["resources/js/app.js", "resources/css/style.scss"],
  bundle: true,
  minify: true,
  logLevel: "debug",
  metafile: true,
  sourcemap: true,
  legalComments: "linked",
  allowOverwrite: true,
  outbase: "resources",
  target: ["es2020", "chrome58", "edge16", "firefox57", "safari11"],
  loader: {
    ".png": "file",
    ".jpg": "file",
    ".svg": "file",
    ".woff": "file",
    ".woff2": "file",
    ".ttf": "file",
    ".eot": "file",
  },
  plugins: [
    sassPlugin({
      basedir: "resources/css",
      loadPaths: ["node_modules", "resources/css"],
      async transform(source) {
        const { css } = await postcss([
          autoprefixer,
          purgeCSSPlugin({
            safelist: [
              "content",
              "is-multiline",
              "is-4by3",
              "is-48x48",
              "hidden-caption",
              "divider",
              "card",
              "image",
              "field",
              "is-grouped",
              "hpt",
              "postcard-editor",
              "icon",
              "is-clickable",
              "close-dialog",
              "is-large",
              "fas",
              "fa-times",
              "fa-2x",
              "mb-2",
              "fa-spinner",
              "fa-pulse",
              "has-sticky-header",
              "is-sticky",
              "is-reversed-mobile",
              "is-clipped",
              "bottom-level",
              "progress-indicator",
              "htmx-request",
              "*-auto",
              "mb-3",
              "mr-*",
              "my-*",
              "mx-*",
              "mb-0",
              "is-one-third-tablet",
              "is-one-quarter-desktop",
              "is-full-mobile",
              "textarea",
              "is-success",
              "is-danger",
            ],
            content: [
              "assets/templ/**/*.templ",
              "assets/templ/**/*.go",
              "resources/js/**/*.js",
              "utils/**/*.go",
            ],
          }),
        ]).process(source, { from: undefined });

        return css;
      },
    }),
    copy({
      resolveFrom: "cwd",
      assets: {
        from: ["./node_modules/htmx.org/dist/htmx.min.js"],
        to: ["./assets/public/js/vendor"],
      },
      watch: true,
    }),
    copy({
      resolveFrom: "cwd",
      assets: {
        from: ["./node_modules/htmx.org/dist/ext/loading-states.js"],
        to: ["./assets/public/js/vendor"],
      },
      watch: true,
    }),
  ],
  outdir: "./assets/public/",
});

fs.writeFileSync("meta.json", JSON.stringify(result.metafile));
