#!/usr/bin/env node
// Launcher for the VS Code Agent Plugin's MCP server. A plugin's mcp.json `command`
// must be a bare name (PATH) or a plugin-relative ./ path — an absolute path isn't
// allowed — and mcp.json has no per-OS command override, so a single command must
// work on macOS, Linux, *and* Windows. A POSIX shell script can't do that (no
// shebang support on Windows), so this is plain Node instead: VS Code's own MCP
// docs use "node" as the portable choice, and it sidesteps the shell entirely.
//
// Binary discovery and auto-download live in ./resolve.js and ./download.js,
// shared with the Copilot agent hook so both behave identically. The binary isn't
// bundled: on first run ensureBinary() downloads the pinned release from GitHub
// (checksum-verified) and caches it under ~/.mantis/bin.
"use strict";

const { spawn } = require("child_process");
const { ensureBinary } = require("./resolve");

(async () => {
  let bin;
  try {
    bin = await ensureBinary();
  } catch (err) {
    console.error(
      "mantis-mem: could not obtain the binary (" +
        String((err && err.message) || err) +
        ").\n" +
        "Fixes: set MANTIS_BIN to an existing binary; or install it with\n" +
        "  go install github.com/oshiruko19/mantis-mem@latest\n" +
        "or download a release from https://github.com/oshiruko19/mantis-mem/releases\n" +
        "and put it on your PATH.",
    );
    process.exit(127);
  }

  const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });

  for (const sig of ["SIGINT", "SIGTERM"]) {
    process.on(sig, () => child.kill(sig));
  }

  child.on("error", (err) => {
    console.error(String(err));
    process.exit(1);
  });

  child.on("exit", (code, signal) => {
    if (signal) process.kill(process.pid, signal);
    else process.exit(code === null ? 1 : code);
  });
})();
