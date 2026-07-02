// Builds the three MV3 bundles (popup, background, content) into dist/,
// then copies manifest.json, popup.html/css, and icons alongside them.
//
// One config, three entrypoints — kept minimal on purpose: MV3 CSP won't
// let us load remote code and won't tolerate eval-based sourcemaps, so
// there's no dev/prod split worth building.

import * as esbuild from "esbuild";
import { readdir, copyFile, mkdir, cp } from "node:fs/promises";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const SRC = join(__dirname, "src");
const OUT = join(__dirname, "dist");

const watch = process.argv.includes("--watch");

const common = {
  bundle: true,
  platform: "browser",
  target: "chrome110",
  format: "iife", // MV3 content scripts + popup: no ESM
  logLevel: "info",
  legalComments: "none",
  minify: !watch,
  sourcemap: watch ? "inline" : false,
};

const entries = {
  "popup.js": join(SRC, "popup", "popup.ts"),
  "content.js": join(SRC, "content", "content.ts"),
  "background.js": join(SRC, "background.ts"),
};

async function copyStatic() {
  await mkdir(OUT, { recursive: true });
  // Manifest, popup HTML/CSS.
  await copyFile(join(SRC, "manifest.json"), join(OUT, "manifest.json"));
  await copyFile(join(SRC, "popup", "popup.html"), join(OUT, "popup.html"));
  await copyFile(join(SRC, "popup", "popup.css"), join(OUT, "popup.css"));

  const icons = join(SRC, "icons");
  if (existsSync(icons)) {
    const files = await readdir(icons);
    if (files.length > 0) {
      await cp(icons, join(OUT, "icons"), { recursive: true });
    }
  }
}

async function build() {
  await copyStatic();
  const results = await Promise.all(
    Object.entries(entries).map(([outfile, entry]) =>
      esbuild.build({
        ...common,
        entryPoints: [entry],
        outfile: join(OUT, outfile),
      })
    )
  );
  const errors = results.flatMap((r) => r.errors);
  if (errors.length > 0) {
    console.error(errors);
    process.exit(1);
  }
  console.log("Built extension/dist/ — load unpacked from there.");
}

async function watchBuild() {
  await copyStatic();
  const contexts = await Promise.all(
    Object.entries(entries).map(([outfile, entry]) =>
      esbuild.context({
        ...common,
        entryPoints: [entry],
        outfile: join(OUT, outfile),
      })
    )
  );
  await Promise.all(contexts.map((c) => c.watch()));
  console.log("Watching src/ — rebuilds on save. Ctrl+C to quit.");
}

if (watch) {
  await watchBuild();
} else {
  await build();
}
