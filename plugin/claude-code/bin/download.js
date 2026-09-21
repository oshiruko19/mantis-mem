// Auto-download the mantis-mem binary from GitHub Releases.
//
// The plugins bundle the MCP launcher, skill, and hooks but NOT the binary — it
// is a per-OS/arch Go build published as a GitHub Release asset by goreleaser.
// This module fetches the release archive that matches the running platform,
// verifies it against the release checksums.txt, extracts the binary, and caches
// it under ~/.mantis/bin so the user never has to build from source or run `make`.
//
// Plain Node, no dependencies. This file is byte-identical in the Claude Code and
// VS Code plugins (plugin/*/bin/download.js) — keep the two copies in sync.
//
// Security: the downloaded binary is executed, so every download is checksum-
// verified (sha256) against the release's checksums.txt and fetched over HTTPS
// from github.com only. Set MANTIS_NO_DOWNLOAD=1 to disable downloading entirely.
"use strict";

const https = require("https");
const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");

const IS_WINDOWS = process.platform === "win32";
const BIN_NAME = IS_WINDOWS ? "mantis-mem.exe" : "mantis-mem";

// Distribution source. Overridable for forks/testing; the version is passed in by
// the caller (read from the plugin manifest) so plugin and binary stay in lockstep.
const REPO = (process.env.MANTIS_REPO || "oshiruko19/mantis-mem").trim();

function envFlag(name) {
  const v = (process.env[name] || "").trim().toLowerCase();
  return v === "1" || v === "true" || v === "yes";
}

function isExecutable(file) {
  try {
    fs.accessSync(file, fs.constants.X_OK);
    return true;
  } catch {
    // X_OK is meaningless on Windows; existence is enough there.
    return IS_WINDOWS && fs.existsSync(file);
  }
}

function managedDir() {
  return path.join(os.homedir(), ".mantis", "bin");
}

// Cache path is versioned so a plugin upgrade re-downloads the matching binary.
function managedPath(version) {
  return path.join(managedDir(), `mantis-mem-${version}${IS_WINDOWS ? ".exe" : ""}`);
}

// Map the Node runtime to goreleaser's os/arch tokens. Returns null on an
// unsupported platform (no published asset).
function platformTokens() {
  const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
  const goarch = { x64: "amd64", arm64: "arm64" }[process.arch];
  return goos && goarch ? { goos, goarch } : null;
}

// Asset names mirror .goreleaser.yaml's archives.name_template exactly:
//   mantis-mem_<version>_<os>_<arch>.tar.gz   (.zip on windows)
function assetName(version, tokens) {
  const ext = tokens.goos === "windows" ? "zip" : "tar.gz";
  return `mantis-mem_${version}_${tokens.goos}_${tokens.goarch}.${ext}`;
}

function assetURL(version, tokens) {
  return `https://github.com/${REPO}/releases/download/v${version}/${assetName(version, tokens)}`;
}

function checksumsURL(version) {
  return `https://github.com/${REPO}/releases/download/v${version}/checksums.txt`;
}

// GET with redirect-following (GitHub redirects release assets to
// objects.githubusercontent.com). Resolves to the final 200 response stream,
// still paused so the caller attaches its own consumers without losing data.
function httpsGet(url, redirectsLeft = 5) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "mantis-mem-launcher" } }, (res) => {
        const { statusCode, headers } = res;
        if (statusCode >= 300 && statusCode < 400 && headers.location) {
          res.resume();
          if (redirectsLeft <= 0) return reject(new Error("too many redirects"));
          const next = new URL(headers.location, url).toString();
          return resolve(httpsGet(next, redirectsLeft - 1));
        }
        if (statusCode !== 200) {
          res.resume();
          return reject(new Error(`GET ${url} -> HTTP ${statusCode}`));
        }
        resolve(res);
      })
      .on("error", reject);
  });
}

// Stream a URL to a file, returning the sha256 of the bytes written.
async function downloadToFile(url, dest) {
  const res = await httpsGet(url);
  const hash = crypto.createHash("sha256");
  await new Promise((resolve, reject) => {
    const out = fs.createWriteStream(dest);
    res.on("data", (c) => hash.update(c));
    res.on("error", reject);
    out.on("error", reject);
    out.on("finish", resolve);
    res.pipe(out);
  });
  return hash.digest("hex");
}

async function downloadText(url) {
  const res = await httpsGet(url);
  const chunks = [];
  await new Promise((resolve, reject) => {
    res.on("data", (c) => chunks.push(c));
    res.on("error", reject);
    res.on("end", resolve);
  });
  return Buffer.concat(chunks).toString("utf8");
}

// Find the expected sha256 for `name` in a goreleaser checksums.txt
// ("<sha256>  <filename>", optionally "*"-prefixed for binary mode).
function checksumFor(checksumsText, name) {
  for (const line of checksumsText.split(/\r?\n/)) {
    const m = line.trim().match(/^([0-9a-fA-F]{64})\s+\*?(.+)$/);
    if (m && path.basename(m[2]) === name) return m[1].toLowerCase();
  }
  return null;
}

// Extract the archive into destDir. tar handles tar.gz everywhere and (bsdtar on
// Windows 10+) zip too; fall back to PowerShell Expand-Archive for zip.
function extract(archivePath, destDir, tokens) {
  if (tokens.goos === "windows") {
    let r = spawnSync("tar", ["-xf", archivePath, "-C", destDir], { stdio: "ignore" });
    if (r.status !== 0) {
      r = spawnSync(
        "powershell",
        [
          "-NoProfile",
          "-Command",
          `Expand-Archive -Force -LiteralPath '${archivePath}' -DestinationPath '${destDir}'`,
        ],
        { stdio: "ignore" },
      );
    }
    if (r.status !== 0) throw new Error("failed to extract zip archive");
  } else {
    const r = spawnSync("tar", ["-xzf", archivePath, "-C", destDir], { stdio: "ignore" });
    if (r.status !== 0) throw new Error("failed to extract tar.gz archive");
  }
}

// Ensure the managed binary for `version` exists, downloading + verifying +
// extracting it on a cache miss. Returns the absolute path, or throws (offline,
// bad checksum, unsupported platform, MANTIS_NO_DOWNLOAD set).
async function ensureManagedBinary(version) {
  const target = managedPath(version);
  if (isExecutable(target)) return target;

  if (envFlag("MANTIS_NO_DOWNLOAD")) {
    throw new Error(
      "no mantis-mem binary found and MANTIS_NO_DOWNLOAD is set (auto-download disabled)",
    );
  }

  const tokens = platformTokens();
  if (!tokens) {
    throw new Error(`unsupported platform ${process.platform}/${process.arch} — no release asset`);
  }
  const name = assetName(version, tokens);

  fs.mkdirSync(managedDir(), { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(managedDir(), ".dl-"));
  try {
    const archivePath = path.join(tmpDir, name);
    const got = await downloadToFile(assetURL(version, tokens), archivePath);
    const want = checksumFor(await downloadText(checksumsURL(version)), name);
    if (!want) throw new Error(`no checksum entry for ${name} in checksums.txt`);
    if (want !== got) throw new Error(`checksum mismatch for ${name}`);

    extract(archivePath, tmpDir, tokens);
    const extracted = path.join(tmpDir, BIN_NAME);
    if (!fs.existsSync(extracted)) throw new Error(`${BIN_NAME} not found in ${name}`);
    if (!IS_WINDOWS) fs.chmodSync(extracted, 0o755);

    // Another concurrent launch may have won the race; if so, keep theirs.
    if (isExecutable(target)) return target;
    try {
      fs.renameSync(extracted, target);
    } catch {
      // Cross-device or existing target: copy instead.
      fs.copyFileSync(extracted, target);
      if (!IS_WINDOWS) fs.chmodSync(target, 0o755);
    }
    return target;
  } finally {
    try {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    } catch {
      /* best-effort cleanup */
    }
  }
}

module.exports = {
  BIN_NAME,
  IS_WINDOWS,
  REPO,
  isExecutable,
  managedDir,
  managedPath,
  platformTokens,
  assetName,
  assetURL,
  checksumsURL,
  ensureManagedBinary,
};
