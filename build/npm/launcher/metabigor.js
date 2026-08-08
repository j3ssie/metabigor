#!/usr/bin/env node
// Unified entry point for the Metabigor CLI distributed via npm.
//
// The platform-specific binary is shipped *gzipped* inside an optional
// dependency package (@j3ssie/metabigor@<version>-<tag>, installed under the
// alias @j3ssie/metabigor-<tag>). On first run we decompress it once into a
// version-scoped cache directory and then exec it, forwarding args, stdio,
// signals and the exit status.

import { spawn } from "node:child_process";
import { createReadStream, createWriteStream, existsSync, statSync } from "node:fs";
import { mkdir, rename, chmod, unlink } from "node:fs/promises";
import { createRequire } from "node:module";
import { pipeline } from "node:stream/promises";
import { createGunzip } from "node:zlib";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

// __dirname / require equivalents in ESM.
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const require = createRequire(import.meta.url);

// Local alias (npm install path) -> nothing else; the underlying package
// published to npm is always @j3ssie/metabigor with a version suffix.
const PLATFORM_PACKAGE_BY_TAG = {
  "linux-x64": "@j3ssie/metabigor-linux-x64",
  "linux-arm64": "@j3ssie/metabigor-linux-arm64",
  "darwin-x64": "@j3ssie/metabigor-darwin-x64",
  "darwin-arm64": "@j3ssie/metabigor-darwin-arm64",
  "win32-x64": "@j3ssie/metabigor-win32-x64",
};

const { platform, arch } = process;

function detectPackageManager() {
  const userAgent = process.env.npm_config_user_agent || "";
  if (/\bbun\//.test(userAgent)) return "bun";
  const execPath = process.env.npm_execpath || "";
  if (execPath.includes("bun")) return "bun";
  return userAgent ? "npm" : null;
}

function reinstallHint() {
  return detectPackageManager() === "bun"
    ? "bun install -g @j3ssie/metabigor@latest"
    : "npm install -g @j3ssie/metabigor@latest";
}

let platformTag = null;
switch (platform) {
  case "linux":
  case "android":
    if (arch === "x64") platformTag = "linux-x64";
    else if (arch === "arm64") platformTag = "linux-arm64";
    break;
  case "darwin":
    if (arch === "x64") platformTag = "darwin-x64";
    else if (arch === "arm64") platformTag = "darwin-arm64";
    break;
  case "win32":
    if (arch === "x64") platformTag = "win32-x64";
    break;
  default:
    break;
}

if (!platformTag) {
  throw new Error(
    `Unsupported platform: ${platform} (${arch}). ` +
      `Metabigor npm builds cover linux/darwin on x64/arm64 and windows on x64. ` +
      `Install from source instead: ` +
      `go install github.com/j3ssie/metabigor/cmd/metabigor@latest`,
  );
}

const platformPackage = PLATFORM_PACKAGE_BY_TAG[platformTag];
const isWindows = platform === "win32";
const binName = isWindows ? "metabigor.exe" : "metabigor";

// Resolve the gzipped binary: prefer the installed optional-dependency
// package, and fall back to the staging layout produced by `make npm-build`
// (build/dist-npm/metabigor-<tag>/vendor/...) so the packages can be smoke
// tested before they are published.
const gzName = "metabigor.gz";
const localCandidates = [
  path.join(__dirname, "..", "vendor", platformTag, gzName),
  path.join(__dirname, "..", "..", `metabigor-${platformTag}`, "vendor", platformTag, gzName),
];

let gzPath = null;
try {
  const pkgJsonPath = require.resolve(`${platformPackage}/package.json`);
  gzPath = path.join(path.dirname(pkgJsonPath), "vendor", platformTag, gzName);
} catch {
  gzPath = localCandidates.find((p) => existsSync(p)) || null;
}

if (!gzPath || !existsSync(gzPath)) {
  throw new Error(
    `Missing platform package ${platformPackage} (the metabigor binary for ` +
      `${platformTag} was not installed). This usually means npm skipped ` +
      `optional dependencies. Reinstall Metabigor:\n    ${reinstallHint()}`,
  );
}

// Version-scoped cache path so upgrades never exec a stale binary, and so the
// extraction directory is always writable by the running user even when the
// package itself was installed into a root-owned global prefix.
const pkgVersion = require("../package.json").version;
const metabigorHome =
  process.env.METABIGOR_HOME || path.join(os.homedir(), ".metabigor");
const binaryDir = path.join(metabigorHome, "npm-bin", pkgVersion, platformTag);
const binaryPath = path.join(binaryDir, binName);

async function ensureBinary() {
  if (existsSync(binaryPath) && statSync(binaryPath).size > 0) {
    return;
  }

  await mkdir(binaryDir, { recursive: true });

  // Decompress to a unique temp file, then atomically rename into place so
  // concurrent first-runs and interrupted extractions can never leave a
  // truncated binary at the final path.
  const tmpPath = path.join(
    binaryDir,
    `metabigor.${process.pid}.${Date.now()}.tmp`,
  );

  try {
    await pipeline(
      createReadStream(gzPath),
      createGunzip(),
      createWriteStream(tmpPath, { mode: 0o755 }),
    );
    await chmod(tmpPath, 0o755);
    await rename(tmpPath, binaryPath);
  } catch (err) {
    await unlink(tmpPath).catch(() => {});
    // Another racing process may have completed the extraction already.
    if (existsSync(binaryPath) && statSync(binaryPath).size > 0) {
      return;
    }
    throw new Error(
      `Failed to unpack the metabigor binary for ${platformTag}: ${err.message}\n` +
        `Try reinstalling:\n    ${reinstallHint()}`,
    );
  }
}

await ensureBinary();

// Use an asynchronous spawn (not spawnSync) so Node can respond to signals
// (e.g. Ctrl-C / SIGINT) while the native binary runs, forward them to the
// child, and mirror the child's termination reason in the parent.
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env,
});

child.on("error", (err) => {
  // eslint-disable-next-line no-console
  console.error(err);
  process.exit(1);
});

const forwardSignal = (signal) => {
  if (child.killed) return;
  try {
    child.kill(signal);
  } catch {
    /* ignore */
  }
};

["SIGINT", "SIGTERM", "SIGHUP"].forEach((sig) => {
  try {
    process.on(sig, () => forwardSignal(sig));
  } catch {
    /* signal not supported on this platform (Windows) */
  }
});

const childResult = await new Promise((resolve) => {
  child.on("exit", (code, signal) => {
    if (signal) {
      resolve({ type: "signal", signal });
    } else {
      resolve({ type: "code", exitCode: code ?? 1 });
    }
  });
});

if (childResult.type === "signal") {
  // Re-emit the same signal so the parent terminates with 128 + n semantics.
  process.kill(process.pid, childResult.signal);
} else {
  process.exit(childResult.exitCode);
}
