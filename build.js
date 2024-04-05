const sourcemap = process.env.NODE_ENV === "development" ? "none" : "external";
const minify = process.env.NODE_ENV === "development" ? false : true;

console.info(`Building Frontend for ${process.env.NODE_ENV} environment`);

console.info("Copying Frontend assets...");

const list = [
  [
    "./node_modules/htmx.org/dist/ext/loading-states.js",
    "./assets/public/js/vendor/loading-states.js",
  ],
];

for await (const [src, dest] of list) {
  console.info(`Copying ${src} to ${dest}`);
  const f = Bun.file(src);
  await Bun.write(f, dest);
}

console.info("Building Frontend...");

const build = await Bun.build({
  entrypoints: ["./resources/js/app.ts"],
  outdir: "./assets/public/js",
  minify,
  sourcemap,
  target: "browser",
  format: "esm",
  splitting: true,
  manifest: true,
});

if (!build.success) {
  console.error("Frontend build failed");
  for (const message of build.logs) {
    // Bun will pretty print the message object
    console.error(message);
  }
} else {
  console.info("Frontend build succeeded");
}
