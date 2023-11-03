import * as esbuild from "esbuild";
import { sassPlugin } from "esbuild-sass-plugin";
import postcss from "postcss";
import autoprefixer from "autoprefixer";
import purgeCSSPlugin from "@fullhuman/postcss-purgecss";
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
            safelist: ["content"],
            content: [
              "assets/views/layout.html",
              "assets/views/pages/**/*.html",
              "assets/views/partials/**/*.html",
              "resources/js/**/*.js",
              "utils/**/*.go",
            ],
          }),
        ]).process(source, { from: undefined });

        return css;
      },
    }),
  ],
  outdir: "./assets/public/",
});

fs.writeFileSync("meta.json", JSON.stringify(result.metafile));
