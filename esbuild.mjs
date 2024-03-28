import * as esbuild from "esbuild";
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
