import * as esbuild from "esbuild";
import { sassPlugin } from "esbuild-sass-plugin";
import postcss from "postcss";
import autoprefixer from "autoprefixer";

esbuild
  .build({
    entryPoints: [
      "./resources/sitebuild/src/js/app.js",
      "./resources/sitebuild/src/css/style.scss",
    ],
    bundle: true,
    minify: true,
    sourcemap: true,
    legalComments: "linked",
    allowOverwrite: true,
    plugins: [
      sassPlugin({
        loadPath: ["./resources/sitebuild/src/css"],
        async transform(source) {
          const { css } = await postcss([autoprefixer]).process(source);
          return css;
        },
      }),
    ],
    outdir: "./assets/public/",
  })
  .then(() => {
    console.log("⚡ Build complete! ⚡");
  })
  .catch(() => process.exit(1));
