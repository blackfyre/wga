import { expect, test } from "bun:test";
import {
  existsSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const root = resolve(import.meta.dir, "../..");

test("release build stages matching maps outside public assets and cleans stale output", () => {
  const fixture = mkdtempSync(join(tmpdir(), "wga-release-"));
  try {
    const scripts = JSON.parse(
      readFileSync(join(root, "package.json"), "utf8"),
    ).scripts;
    writeFileSync(join(fixture, "package.json"), JSON.stringify({ scripts }));
    mkdirSync(join(fixture, "resources/js"), { recursive: true });
    writeFileSync(
      join(fixture, "resources/js/app.ts"),
      'console.log("release fixture");',
    );
    for (const dir of ["dist/browser-assets/js", "internal/assets/public/js"]) {
      mkdirSync(join(fixture, dir), { recursive: true });
      writeFileSync(join(fixture, dir, "stale.js.map"), "stale");
    }
    const result = Bun.spawnSync([process.execPath, "run", "build:js"], {
      cwd: fixture,
    });
    expect(result.exitCode).toBe(0);
    const staged = readdirSync(join(fixture, "dist/browser-assets/js"));
    const published = readdirSync(join(fixture, "internal/assets/public/js"));
    expect(staged).toContain("app.js.map");
    expect(staged).not.toContain("stale.js.map");
    expect(published.filter((name) => name.endsWith(".map"))).toEqual([]);
    expect(
      readFileSync(join(fixture, "internal/assets/public/js/app.js")),
    ).toEqual(readFileSync(join(fixture, "dist/browser-assets/js/app.js")));
    const map = JSON.parse(
      readFileSync(join(fixture, "dist/browser-assets/js/app.js.map"), "utf8"),
    );
    expect(map.sourcesContent).toContain('console.log("release fixture");');
    const dockerfile = readFileSync(join(root, "Dockerfile"), "utf8");
    expect(dockerfile).toContain(
      "/app/src/dist/browser-assets/js /usr/local/share/wga/sourcemaps",
    );
    expect(dockerfile).toContain(
      "internal/buildinfo.Version=${RAILWAY_GIT_COMMIT_SHA}",
    );
    expect(dockerfile).not.toContain("ARG SENTRY_AUTH_TOKEN");
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});

test("entrypoint uploads stable release inputs before serving and fails closed", () => {
  const fixture = mkdtempSync(join(tmpdir(), "wga-entrypoint-"));
  try {
    const entrypoint = readFileSync(
      join(root, "resources/scripts/start-wga.sh"),
      "utf8",
    ).replaceAll("/usr/local/share/wga", fixture);
    writeFileSync(join(fixture, "start"), entrypoint);
    writeFileSync(join(fixture, "release"), "fixture-commit\n");
    mkdirSync(join(fixture, "sourcemaps"));
    writeFileSync(
      join(fixture, "sentry-cli"),
      '#!/bin/sh\n[ "$SENTRY_AUTH_TOKEN" = "fixture-secret" ] || exit 2\nprintf "%s\\n" "$@" > "$FIXTURE/upload"\nexit "${UPLOAD_STATUS:-0}"\n',
      { mode: 0o755 },
    );
    writeFileSync(
      join(fixture, "app"),
      '#!/bin/sh\n[ -z "${SENTRY_AUTH_TOKEN+x}" ] || exit 3\nprintf started > "$FIXTURE/started"\n',
      { mode: 0o755 },
    );
    const env = {
      PATH: `${fixture}:${process.env.PATH}`,
      FIXTURE: fixture,
      WGA_SENTRY_BROWSER_DSN: "fixture-dsn",
      SENTRY_AUTH_TOKEN: "fixture-secret",
    };
    const run = (overrides = {}) =>
      Bun.spawnSync(["sh", join(fixture, "start"), join(fixture, "app")], {
        env: { ...env, ...overrides },
      });
    expect(run().exitCode).toBe(0);
    const inputs = readFileSync(join(fixture, "upload"), "utf8");
    expect(inputs.split("\n")).toEqual([
      "sourcemaps",
      "upload",
      "--org",
      "web-gallery-of-art-modernisation",
      "--project",
      "wga-browser",
      "--release",
      "fixture-commit",
      "--url-prefix",
      "~/assets/js",
      "--validate",
      "--wait",
      join(fixture, "sourcemaps"),
      "",
    ]);
    expect(inputs).not.toContain("fixture-secret");
    expect(run().exitCode).toBe(0);
    expect(readFileSync(join(fixture, "upload"), "utf8")).toBe(inputs);
    rmSync(join(fixture, "started"));
    expect(run({ UPLOAD_STATUS: "1" }).exitCode).toBe(1);
    expect(existsSync(join(fixture, "started"))).toBe(false);
    expect(run({ SENTRY_AUTH_TOKEN: "" }).exitCode).not.toBe(0);
    expect(
      run({ WGA_SENTRY_BROWSER_DSN: "", SENTRY_AUTH_TOKEN: "" }).exitCode,
    ).toBe(0);
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});
