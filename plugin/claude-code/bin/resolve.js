// Shared resolution logic for the Claude Code plugin: locate the mantis-mem
// binary and determine the project working directory. Used by both the MCP
// launcher (mantis-mem-launcher.js) and the SessionStart hook (session-start.js)
// so binary discovery and project scoping stay identical between them.
"use strict";

const { spawnSync } = require("child_process");
const { existsSync, accessSync, constants, statSync } = require("fs");
const os = require("os");
const path = require("path");
const download = require("./download");

const IS_WINDOWS = process.platform === "win32";
const BIN_NAME = IS_WINDOWS ? "mantis-mem.exe" : "mantis-mem";

function isExecutable(file) {
  try {
    accessSync(file, constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function candidateIn(dir) {
  const candidate = path.join(dir, BIN_NAME);
  return existsSync(candidate) && isExecutable(candidate) ? candidate : null;
}

function findOnPath() {
  const dirs = (process.env.PATH || "").split(path.delimiter).filter(Boolean);
  for (const dir of dirs) {
    const hit = candidateIn(dir);
    if (hit) return hit;
  }
  return null;
}

function findInKnownDirs() {
  const dirs = [path.join(os.homedir(), "go", "bin")];
  if (!IS_WINDOWS) dirs.push("/usr/local/bin", "/opt/homebrew/bin");
  for (const dir of dirs) {
    const hit = candidateIn(dir);
    if (hit) return hit;
  }
  return null;
}

function goEnv(name) {
  const result = spawnSync("go", ["env", name], {
    encoding: "utf8",
    shell: IS_WINDOWS,
  });
  return result.status === 0 ? result.stdout.trim() : "";
}

function findViaGoEnv() {
  const dirs = [];
  const gobin = goEnv("GOBIN");
  if (gobin) dirs.push(gobin);
  const gopath = goEnv("GOPATH");
  if (gopath) dirs.push(path.join(gopath, "bin"));
  for (const dir of dirs) {
    const hit = candidateIn(dir);
    if (hit) return hit;
  }
  return null;
}

// findBinary returns the absolute path to the mantis-mem binary, checking PATH,
// the common install directories, and `go env` — GUI-launched editors often
// don't inherit ~/go/bin on PATH. Returns null if it can't be found.
function findBinary() {
  return findOnPath() || findInKnownDirs() || findViaGoEnv();
}

// pluginVersion is the version the binary is pinned to — read from the plugin
// manifest so plugin and binary stay in lockstep (overridable via MANTIS_VERSION).
function pluginVersion() {
  const override = (process.env.MANTIS_VERSION || "").trim().replace(/^v/, "");
  if (override) return override;
  try {
    const manifest = require(path.join(__dirname, "..", ".claude-plugin", "plugin.json"));
    return manifest && manifest.version ? String(manifest.version) : null;
  } catch {
    return null;
  }
}

// explicitBin honors MANTIS_BIN — an escape hatch to point the plugin at a
// specific binary (a `go build` / `make install` dev build, or an air-gapped copy).
function explicitBin() {
  const bin = (process.env.MANTIS_BIN || "").trim();
  return bin && existsSync(bin) ? bin : null;
}

// managedIfPresent returns the cached auto-downloaded binary if it already exists,
// without downloading anything.
function managedIfPresent() {
  const version = pluginVersion();
  if (!version) return null;
  const p = download.managedPath(version);
  return download.isExecutable(p) ? p : null;
}

// findResolved is the find-only resolution used by the hooks: explicit override ->
// local install (PATH / known dirs / go env) -> cached download. Never downloads.
function findResolved() {
  return explicitBin() || findBinary() || managedIfPresent();
}

// ensureBinary is the launcher's resolution: find-only first (so a local dev build
// still wins), otherwise download the pinned release binary and cache it. Async
// because a cache miss fetches from GitHub Releases. Rejects if nothing works.
async function ensureBinary() {
  const found = findResolved();
  if (found) return found;
  const version = pluginVersion();
  if (!version) {
    throw new Error("could not determine mantis-mem version from the plugin manifest");
  }
  return download.ensureManagedBinary(version);
}

// projectCwd returns CLAUDE_PROJECT_DIR when it's set to a real directory, else
// undefined. mantis-mem resolves its project from the working directory (nearest
// ancestor with .git); Claude Code sets CLAUDE_PROJECT_DIR to the stable project
// root, so running the binary there scopes memory to the open project rather than
// the plugin install directory.
function projectCwd() {
  const dir = (process.env.CLAUDE_PROJECT_DIR || "").trim();
  if (!dir) return undefined;
  try {
    return statSync(dir).isDirectory() ? dir : undefined;
  } catch {
    return undefined;
  }
}

module.exports = {
  findBinary,
  findResolved,
  ensureBinary,
  pluginVersion,
  projectCwd,
  BIN_NAME,
};

// CLI: used by the SessionStart hook to locate the binary through the same logic
// as the launcher. `--path` is find-only (prints the resolved path or exits 1,
// never downloads); `--print-url` reports the release asset URL (verification).
if (require.main === module) {
  const arg = process.argv[2] || "--path";
  if (arg === "--path") {
    const bin = findResolved();
    if (bin) {
      process.stdout.write(bin);
      process.exit(0);
    }
    process.exit(1);
  } else if (arg === "--print-url") {
    const version = pluginVersion();
    const tokens = download.platformTokens();
    if (version && tokens) {
      process.stdout.write(download.assetURL(version, tokens) + "\n");
      process.exit(0);
    }
    process.exit(1);
  } else if (arg === "--ensure") {
    ensureBinary().then(
      (bin) => {
        process.stdout.write(bin);
        process.exit(0);
      },
      (err) => {
        process.stderr.write(String((err && err.message) || err) + "\n");
        process.exit(1);
      },
    );
  } else {
    process.stderr.write(`unknown flag: ${arg}\n`);
    process.exit(2);
  }
}
