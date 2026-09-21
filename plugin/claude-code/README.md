# mantis-mem — Claude Code plugin

This folder is a [Claude Code plugin](https://code.claude.com/docs/en/plugins). Installing
it bundles:

- **`.mcp.json`** — the mantis-mem MCP server (`mem_*` tools), started for you when the
  plugin is enabled.
- **`skills/mantis-mem/SKILL.md`** — the usage skill that tells Claude how and when to use
  the `mem_*` tools.
- **`hooks/hooks.json`** — a `SessionStart` hook (native shell scripts) that deterministically
  injects the usage contract and this project's recent memory at session start and after
  compaction.

```
plugin/claude-code/
  .claude-plugin/plugin.json     # plugin manifest (name, version, author)
  .mcp.json                      # MCP server (command -> node ./bin/mantis-mem-launcher.js serve)
  bin/mantis-mem-launcher.js     # resolves/downloads the binary + scopes memory to the project
  bin/resolve.js                 # shared binary resolution (find local, else the cached download)
  bin/download.js                # fetch + checksum-verify the release binary from GitHub
  hooks/hooks.json               # SessionStart / post-compaction orientation
  hooks/session-start.sh         # macOS / Linux / Git Bash — injects contract + recent memory
  hooks/session-start.ps1        # Windows PowerShell equivalent
  skills/mantis-mem/SKILL.md     # the "use mantis-mem" skill
```

### Why the hook is a shell script but the launcher is Node

`hooks.json` runs its `command` through a shell (`bash` by default; Git Bash or PowerShell
on Windows), so the hook can be a native `.sh` with a `.ps1` fallback — no Node. `.mcp.json`,
by contrast, has a **single** `command` with no per-OS branch, so the MCP server launcher
must be one file that runs everywhere; Node is the portable choice there. (See the Windows
note under *Prerequisites* to wire up the `.ps1` on PowerShell-only Windows.)

## Skill vs. hook (two layers)

The plugin orients the agent two ways, on purpose:

- The **skill** is *advisory* — it teaches Claude when to reach for the `mem_*` tools, but
  triggers softly (the model may not always invoke it).
- The **`SessionStart` hook** is *deterministic* — it runs `mantis-mem context` and injects
  the usage contract plus this project's recent memory into every session (and again after
  compaction, so memory survives context loss). The hook is **read-only**: it never writes
  to memory. Saving stays deliberate, driven by the agent through `mem_save` /
  `mem_session_summary`. This is intentional — mantis-mem is a **curated** store, not a
  transcript sink, so there is no passive/auto capture on every prompt.

## Prerequisites

1. **Node.js** on PATH — `.mcp.json` has no per-OS command override, so the launcher is
   plain Node (not a shell script) to run unmodified on macOS, Linux, and Windows. Claude
   Code already requires Node, so this is normally already satisfied.
2. **The mantis-mem binary — fetched automatically.** The launcher
   (`bin/mantis-mem-launcher.js`) resolves `mantis-mem` (`mantis-mem.exe` on Windows) in this
   order: `MANTIS_BIN`, then a local install (`~/go/bin`, `/usr/local/bin`,
   `/opt/homebrew/bin`, PATH, or `go env`), then a previously cached download, and finally it
   **downloads** the matching release from GitHub (checksum-verified) into `~/.mantis/bin` — so
   **no build, PATH change, or symlink is needed**. To use your own build instead, put it in
   one of those dirs or set `MANTIS_BIN`:
   ```bash
   make install        # optional: -> ~/go/bin/mantis-mem (preferred over the download)
   ```
   Set `MANTIS_NO_DOWNLOAD=1` to disable the download (air-gapped). Details:
   [docs/install.md](../../docs/install.md#how-the-binary-is-obtained).
3. **Windows only (optional).** The hook runs `hooks/session-start.sh`, which works on
   macOS, Linux, and Windows **with Git Bash** (Claude Code's default hook shell). On a
   Windows machine *without* Git Bash, point the hook at the bundled PowerShell version:
   in `hooks/hooks.json`, add `"shell": "powershell"` to each hook entry and change the
   command to `"& '${CLAUDE_PLUGIN_ROOT}/hooks/session-start.ps1'"`.

## Install

From the repo (which doubles as a plugin marketplace):

```
/plugin marketplace add oshiruko19/mantis-mem
/plugin install mantis-mem@mantis-mem
```

See **[docs/install.md](../../docs/install.md)** for the full walkthrough (including local
checkout and settings-based installs) and the **How to use in claude-code** section.

## How memory is scoped

mantis-mem keeps a separate memory per project, resolved from the server's working
directory (nearest ancestor containing `.git`). Claude Code sets `CLAUDE_PROJECT_DIR` in
the server's environment; the launcher runs the binary with that as its working directory,
so memory follows the open project rather than the plugin install directory. Confirm the
resolved project at any time with the `mem_current_project` tool.

State lives in one SQLite file (`~/.mantis/mantis_mem.db` by default; override with an
absolute `MANTIS_DB` path).
