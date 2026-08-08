#!/usr/bin/env node
// Stage the @j3ssie/metabigor npm packages.
//
// Produces, under build/dist-npm/:
//   - metabigor/             the thin launcher package (@j3ssie/metabigor@<v>)
//   - metabigor-<tag>/       platform packages (@j3ssie/metabigor@<v>-<tag>)
//                            each carrying the gzipped binary in vendor/<tag>/
//
// One npm name, version-suffixed platform builds (codex-style): the launcher
// pulls its platform's binary in as an optionalDependency aliased to
// `npm:@j3ssie/metabigor@<v>-<tag>`, so npm only downloads the one build that
// matches the host.
//
// The binary is gzipped so each platform package stays small on the registry;
// bin/metabigor.js decompresses it once on first run.
//
// Source binaries come from goreleaser output in dist/
// (metabigor_<goos>_<goarch>[_<variant>]/metabigor). Run `make snapshot` first.
//
// Usage:
//   node build/npm/build.mjs [--pack] [--allow-missing=<tag,tag>]
//   METABIGOR_VERSION=3.0.1 node build/npm/build.mjs

import { spawnSync } from "node:child_process";
import {
  createWriteStream,
  existsSync,
  readdirSync,
  readFileSync,
  rmSync,
  mkdirSync,
  copyFileSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { Readable } from "node:stream";
import { pipeline } from "node:stream/promises";
import { createGzip } from "node:zlib";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, "..", "..");
const DIST_DIR = path.join(REPO_ROOT, "dist");
const OUT_DIR = path.join(REPO_ROOT, "build", "dist-npm");
const LAUNCHER_SRC = path.join(__dirname, "launcher", "metabigor.js");
// Bundle the full project README onto the npm package page.
const README_SRC = path.join(REPO_ROOT, "README.md");
const LICENSE_SRC = path.join(REPO_ROOT, "LICENSE");
const CONSTANTS_GO = path.join(REPO_ROOT, "internal", "core", "constants.go");

const NPM_NAME = "@j3ssie/metabigor";
const BIN_NAME = "metabigor";
const LICENSE_ID = "MIT";
const HOMEPAGE = "https://github.com/j3ssie/metabigor";
const DESCRIPTION = "Metabigor - OSINT power without API key hassle";
const KEYWORDS = [
  "metabigor",
  "osint",
  "recon",
  "reconnaissance",
  "security",
  "asn",
  "certificate-transparency",
  "subdomain",
  "cidr",
  "bugbounty",
];
const REPOSITORY = {
  type: "git",
  url: "git+https://github.com/j3ssie/metabigor.git",
};
const ENGINES = { node: ">=16" };

const PLATFORMS = [
  { tag: "linux-x64", goos: "linux", goarch: "amd64", os: "linux", cpu: "x64" },
  { tag: "linux-arm64", goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64" },
  { tag: "darwin-x64", goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64" },
  { tag: "darwin-arm64", goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64" },
  { tag: "win32-x64", goos: "windows", goarch: "amd64", os: "win32", cpu: "x64", exe: true },
];

const args = process.argv.slice(2);
const doPack = args.includes("--pack");
const allowMissing = new Set(
  (args.find((a) => a.startsWith("--allow-missing=")) || "")
    .split("=")[1]
    ?.split(",")
    .map((s) => s.trim())
    .filter(Boolean) || [],
);

function fail(msg) {
  console.error(`\x1b[31m[!] ${msg}\x1b[0m`);
  process.exit(1);
}

function info(msg) {
  console.log(`\x1b[36m[*]\x1b[0m ${msg}`);
}

// --- version --------------------------------------------------------------

function deriveBaseVersion() {
  if (process.env.METABIGOR_VERSION) {
    return process.env.METABIGOR_VERSION.replace(/^v/, "");
  }
  const src = readFileSync(CONSTANTS_GO, "utf8");
  const m = src.match(/^\s*VERSION\s*=\s*"v?([^"]+)"/m);
  if (!m) fail(`Could not parse VERSION from ${CONSTANTS_GO}`);
  return m[1];
}

const baseVersion = deriveBaseVersion();
if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(baseVersion)) {
  console.warn(
    `\x1b[33m[warn] "${baseVersion}" does not look like a semver string; npm may reject it.\x1b[0m`,
  );
}
const platformVersion = (tag) => `${baseVersion}-${tag}`;

// --- built-version verification -------------------------------------------

// goreleaser names each archive `metabigor_v<version>_<os>_<arch>.tar.gz` from
// the version it actually built (archives.name_template in .goreleaser.yaml)
// and `--clean` wipes dist/ at the start of every run — so the archive name is
// an authoritative record of what the unpacked binaries in dist/ really are.
//
// Reading the version out of the binary itself is NOT reliable: a build embeds
// its own docs and skill files, which mention other version strings. Refuse to
// pack whenever dist/ holds binaries built for a version other than the one
// being published — npm versions are immutable, so shipping a stale binary
// under a fresh version number cannot be undone.
function verifyReleaseVersion(expected) {
  if (!existsSync(DIST_DIR)) return; // findSourceBinary fails later with a clearer message
  const archiveRe =
    /^metabigor_v(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)_(?:linux|darwin|windows)_[0-9a-z]+\.(?:tar\.gz|zip)$/;
  const built = new Set();
  for (const f of readdirSync(DIST_DIR)) {
    const m = f.match(archiveRe);
    if (m) built.add(m[1]);
  }
  if (built.size === 0) {
    // Fail closed: dist/ exists but holds no goreleaser archives to verify
    // against, so the unpacked binaries there are unverifiable and may be left
    // over from an older build.
    fail(
      `cannot verify the built version against ${expected}: no goreleaser ` +
        `archives (metabigor_v<version>_<os>_<arch>.tar.gz) in ${DIST_DIR}. The ` +
        `unpacked binaries there are unverifiable and may be STALE — packaging ` +
        `them risks shipping a binary that reports the wrong version. Run ` +
        `\`make snapshot\` to rebuild for ${expected} before packing.`,
    );
  }
  if (built.size !== 1 || !built.has(expected)) {
    fail(
      `built-version mismatch: dist/ was built for [${[...built].sort().join(", ")}] ` +
        `but this npm publish is version ${expected}. The binaries in dist/ are STALE — ` +
        `packaging them would ship a binary whose \`metabigor version\` reports the ` +
        `wrong number. Run \`make snapshot\` to rebuild for ${expected} before packing.`,
    );
  }
  info(`verified dist/ binaries were built for ${expected}`);
}

// --- locate goreleaser binaries -------------------------------------------

function findSourceBinary(p) {
  if (!existsSync(DIST_DIR)) {
    fail(
      `${DIST_DIR} not found. Build cross-platform binaries first ` +
        `(e.g. \`make snapshot\`).`,
    );
  }
  const binFile = p.exe ? `${BIN_NAME}.exe` : BIN_NAME;
  // goreleaser appends a microarchitecture variant to some dirs
  // (metabigor_linux_amd64_v1), so match an optional trailing segment.
  const re = new RegExp(`^${BIN_NAME}_${p.goos}_${p.goarch}(?:_.+)?$`);
  const dirs = readdirSync(DIST_DIR)
    .filter((d) => re.test(d))
    .filter((d) => existsSync(path.join(DIST_DIR, d, binFile)))
    .sort();
  if (dirs.length === 0) return null;
  if (dirs.length > 1) {
    console.warn(
      `\x1b[33m[warn] multiple goreleaser dirs for ${p.goos}/${p.goarch}: ` +
        `${dirs.join(", ")} — using ${dirs[0]}\x1b[0m`,
    );
  }
  return path.join(DIST_DIR, dirs[0], binFile);
}

// --- staging --------------------------------------------------------------

function writeJson(file, obj) {
  mkdirSync(path.dirname(file), { recursive: true });
  writeFileSync(file, JSON.stringify(obj, null, 2) + "\n");
}

function humanSize(bytes) {
  return `${(bytes / 1048576).toFixed(0)} MB`;
}

async function gzipBuffer(buf, dest) {
  mkdirSync(path.dirname(dest), { recursive: true });
  await pipeline(
    Readable.from([buf]),
    createGzip({ level: 9 }),
    createWriteStream(dest),
  );
}

async function stagePlatformPackage(p) {
  const src = findSourceBinary(p);
  if (!src) {
    if (allowMissing.has(p.tag)) {
      console.warn(
        `\x1b[33m[warn] skipping ${p.tag}: no binary for ${p.goos}/${p.goarch}\x1b[0m`,
      );
      return false;
    }
    fail(
      `Missing binary for ${p.goos}/${p.goarch} (tag ${p.tag}). ` +
        `Run \`make snapshot\` or pass --allow-missing=${p.tag}.`,
    );
  }

  const binBuf = readFileSync(src);
  const pkgDir = path.join(OUT_DIR, `${BIN_NAME}-${p.tag}`);
  const gzPath = path.join(pkgDir, "vendor", p.tag, `${BIN_NAME}.gz`);

  info(`packaging ${p.tag} (${humanSize(binBuf.length)} -> gzip)`);
  await gzipBuffer(binBuf, gzPath);

  writeJson(path.join(pkgDir, "package.json"), {
    name: NPM_NAME,
    version: platformVersion(p.tag),
    description: `${DESCRIPTION} (${p.tag} prebuilt binary)`,
    license: LICENSE_ID,
    homepage: HOMEPAGE,
    repository: REPOSITORY,
    os: [p.os],
    cpu: [p.cpu],
    engines: ENGINES,
    files: ["vendor"],
  });
  if (existsSync(README_SRC)) copyFileSync(README_SRC, path.join(pkgDir, "README.md"));
  if (existsSync(LICENSE_SRC)) copyFileSync(LICENSE_SRC, path.join(pkgDir, "LICENSE"));

  console.log(`    -> ${pkgDir}  (gzipped ${humanSize(statSync(gzPath).size)})`);
  return true;
}

function stageMainPackage(stagedTags) {
  const pkgDir = path.join(OUT_DIR, BIN_NAME);
  const binDir = path.join(pkgDir, "bin");
  mkdirSync(binDir, { recursive: true });
  copyFileSync(LAUNCHER_SRC, path.join(binDir, `${BIN_NAME}.js`));
  if (existsSync(README_SRC)) copyFileSync(README_SRC, path.join(pkgDir, "README.md"));
  if (existsSync(LICENSE_SRC)) copyFileSync(LICENSE_SRC, path.join(pkgDir, "LICENSE"));

  const optionalDependencies = {};
  for (const p of PLATFORMS) {
    if (!stagedTags.has(p.tag)) continue;
    optionalDependencies[`${NPM_NAME}-${p.tag}`] =
      `npm:${NPM_NAME}@${platformVersion(p.tag)}`;
  }

  writeJson(path.join(pkgDir, "package.json"), {
    name: NPM_NAME,
    version: baseVersion,
    description: DESCRIPTION,
    keywords: KEYWORDS,
    license: LICENSE_ID,
    homepage: HOMEPAGE,
    repository: REPOSITORY,
    type: "module",
    bin: { [BIN_NAME]: `bin/${BIN_NAME}.js` },
    engines: ENGINES,
    files: ["bin"],
    optionalDependencies,
  });
  info(`staged main package -> ${pkgDir}`);
}

function npmPack(pkgDir) {
  const res = spawnSync(
    "npm",
    ["pack", "--json", "--pack-destination", OUT_DIR],
    { cwd: pkgDir, encoding: "utf8" },
  );
  if (res.status !== 0) {
    fail(`npm pack failed in ${pkgDir}:\n${res.stderr || res.stdout}`);
  }
  try {
    const out = JSON.parse(res.stdout);
    const name = out[0]?.filename;
    if (name) console.log(`    -> ${path.join(OUT_DIR, name)}`);
  } catch {
    /* non-fatal: tarball is still written */
  }
}

// --- main -----------------------------------------------------------------

info(`metabigor npm build — version ${baseVersion}`);
verifyReleaseVersion(baseVersion);
if (existsSync(OUT_DIR)) rmSync(OUT_DIR, { recursive: true, force: true });
mkdirSync(OUT_DIR, { recursive: true });

const stagedTags = new Set();
for (const p of PLATFORMS) {
  if (await stagePlatformPackage(p)) stagedTags.add(p.tag);
}
if (stagedTags.size === 0) fail("No platform packages were staged.");
stageMainPackage(stagedTags);

if (doPack) {
  info("running npm pack on each staged package...");
  npmPack(path.join(OUT_DIR, BIN_NAME));
  for (const tag of stagedTags) npmPack(path.join(OUT_DIR, `${BIN_NAME}-${tag}`));
}

info("done.");
console.log(`\nStaged in ${OUT_DIR}:`);
console.log(`  ${NPM_NAME}@${baseVersion}  (main)`);
for (const tag of stagedTags) {
  console.log(`  ${NPM_NAME}@${platformVersion(tag)}  (${tag})`);
}
console.log(
  `\nVerify:  node ${path.join(OUT_DIR, BIN_NAME, "bin", `${BIN_NAME}.js`)} version`,
);
