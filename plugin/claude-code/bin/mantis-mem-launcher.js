#!/usr/bin/env node
// Launcher for the Claude Code plugin's MCP server. A plugin's .mcp.json has no
// per-OS command override and no `cwd` field for stdio servers, so this plain-Node
// launcher does two jobs a bare command can't:
//
//   1. Find the binary. mantis-mem is a Go binary the user installs themselves
//      (`make install` -> ~/go/bin); GUI-launched editors often don't inherit
//      `~/go/bin` on PATH. Node is the portable choice (works unmodified on
//      macOS, Linux, and Windows; Claude Code already requires Node).
//
//   2. Scope memory to the project. mantis-mem resolves its project from the
//      server's working directory (nearest ancestor with .git). Claude Code sets
//      CLAUDE_PROJECT_DIR in the spawned server's environment, so this runs the
//      binary with that as its cwd — memory follows the open project, not the
//      plugin install directory.
//
// Binary discovery, auto-download, and project scoping live in ./resolve.js and
// ./download.js, shared with the SessionStart hook so both behave identically. The
// binary isn't bundled: on first run ensureBinary() downloads the pinned release
// from GitHub (checksum-verified) and caches it under ~/.mantis/bin.
"use strict";

const { spawn } = require("child_process");
const { ensureBinary, projectCwd } = require("./resolve");

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

  const child = spawn(bin, process.argv.slice(2), {
    stdio: "inherit",
    cwd: projectCwd(),
  });

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
