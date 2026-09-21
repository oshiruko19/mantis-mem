# mantis-mem — VS Code Agent Plugin

This folder is a [VS Code **Agent Plugin**](https://code.visualstudio.com/docs/agent-customization/agent-plugins)
([Agent Plugins 1.0](https://agent-plugins.org/)). Enabling it bundles:

- **`mcp.json`** — the mantis-mem MCP server (auto-starts when the plugin is enabled).
- **`skills/mantis-mem/SKILL.md`** — the usage skill that tells Copilot how/when to use
  the `mem_*` tools.
- **`com.github.copilot/hooks/hooks.json`** — a Copilot **agent hook** (Preview) that
  deterministically injects the `mem_*` usage contract and this workspace's recent memory
  at session start and before compaction.

```
plugin/vscode/
  plugin.json                                    # Agent Plugins 1.0 manifest ($schema, name)
  mcp.json                                       # MCP server (command → node ./bin/mantis-mem-launcher.js serve)
  bin/mantis-mem-launcher.js                     # resolves/downloads the binary regardless of VS Code's PATH
  bin/resolve.js                                 # shared binary resolution (find local, else the cached download)
  bin/download.js                                # fetch + checksum-verify the release binary from GitHub
  com.github.copilot/hooks/hooks.json            # Copilot SessionStart / PreCompact hooks (bash + powershell)
  com.github.copilot/hooks/mantis-mem-hook.sh    # macOS / Linux — emits contract + recent memory as JSON
  com.github.copilot/hooks/mantis-mem-hook.ps1   # Windows PowerShell equivalent
  skills/mantis-mem/SKILL.md                     # the "use mantis-mem" skill
```

The hook config uses Copilot's per-OS command keys — `bash` runs the `.sh`, `powershell`
runs the `.ps1` — so the hook needs no Node. (The MCP **server** launcher stays Node
because `mcp.json` has only one `command` with no per-OS branch; see *Caveats*.)

## Three layers of orientation

The plugin nudges Copilot to use mantis-mem at three strengths, weakest to strongest:

1. **Skill** (`skills/mantis-mem/SKILL.md`) — advisory; loads only when its description
   matches the task.
2. **Repo instructions** ([`.github/copilot-instructions.md`](../../.github/copilot-instructions.md))
   — *always* applied to every request, but only **static** text (the contract; it can't
   run anything).
3. **Agent hook** (`com.github.copilot/hooks/hooks.json`) — the deterministic, **dynamic**
   layer: it runs `mantis-mem context` and injects the contract **plus this workspace's
   recent memory** into every session (and before compaction, so context survives). This
   is the VS Code equivalent of the Claude Code plugin's `SessionStart` hook.

The hook is **read-only** — it never writes memory. Saving stays deliberate via `mem_save`
/ `mem_session_summary`; there is no passive/auto capture on every prompt (mantis-mem is a
*curated* store, not a transcript sink).

## Prerequisites

1. **Node.js** on PATH — mcp.json has no per-OS command override, so the launcher is
   plain Node (not a shell script) to work unmodified on macOS, Linux, and Windows.
2. **The binary — fetched automatically.** The launcher (`./bin/mantis-mem-launcher.js`)
   resolves `mantis-mem` (`mantis-mem.exe` on Windows) in order: `MANTIS_BIN`, a local install
   (`~/go/bin`, `/usr/local/bin`, `/opt/homebrew/bin`, PATH, or `go env`), a cached download,
   then a **checksum-verified download** of the matching release from GitHub into
   `~/.mantis/bin` — so **no build, PATH change, or symlink is needed** (VS Code's GUI PATH
   doesn't matter). To use your own build, put it in one of those dirs or set `MANTIS_BIN`; set
   `MANTIS_NO_DOWNLOAD=1` to disable the download. Optional build:
   ```bash
   make install        # optional: -> ~/go/bin/mantis-mem (preferred over the download)
   ```
3. **Enable agent plugins** (they're experimental): set **`chat.plugins.enabled`** to
   `true` in Settings.
4. **Enable agent hooks** (Preview) — for the orientation hook to run, set
   **`chat.useCustomAgentHooks`** to `true`. Without it the MCP server and skill still
   work; you just lose the deterministic session-start memory injection.

## Install (local plugin)

Register this folder with the **`chat.pluginLocations`** setting (Settings → JSON):

```json
{
  "chat.plugins.enabled": true,
  "chat.pluginLocations": {
    "/Users/aatrari/Documents/shiruko/mantis-mem/plugin/vscode": true
  }
}
```

`true` enables it. It then appears under **Extensions view → “Agent Plugins –
Installed”**, where you can enable/disable it globally or per workspace. (You can also
manage plugins from the Chat view: gear icon → **Plugins**.)

> **Install from source (git URL)** — VS Code's **Chat: Install Plugin From Source**
> expects `plugin.json` at the **repo root**. This plugin lives in a subdirectory
> (`plugin/vscode/`), so use the local-directory method above rather than the repo URL.

## Verify

1. **MCP: List Servers** (Command Palette) → `mantis-mem` is listed and **Running**.
2. Chat (`⌃⌘I`) → **Agent** mode → **Configure Tools** (🛠): the 8 `mem_*` tools appear.
3. **Chat: Configure Skills** lists the `mantis-mem` skill (from the plugin).
4. Ask the agent to _"`mem_save` a note then `mem_search` it"_ and confirm the round-trip.
5. **Hook** (needs `chat.useCustomAgentHooks`): start a fresh chat session in a repo that
   has memory; the agent should open already aware of the recent memory (latest session
   summary + recent observations) without being asked. You can sanity-check the hook script
   itself from a terminal — it follows Copilot's stdin/stdout contract:
   ```bash
   printf '{"hook_event_name":"SessionStart","cwd":"'"$PWD"'"}' \
     | bash plugin/vscode/com.github.copilot/hooks/mantis-mem-hook.sh
   ```
   It should print JSON with a populated `hookSpecificOutput.additionalContext` (or
   `{"continue":true}` if there's no memory yet). On Windows, run the `.ps1` the same way.

## Troubleshooting (from the VS Code docs)

- **Plugin doesn't appear** → confirm `chat.plugins.enabled` is `true`, and that the
  path in `chat.pluginLocations` points at this folder (the one containing `plugin.json`).
- **MCP server won't start / no `mem_*` tools** → either Node isn't on PATH (the launcher and
  the binary download both run under `node`), or the first-run download failed. Check
  connectivity to `github.com`; as a fallback set `MANTIS_BIN` to a manually downloaded binary,
  or place `mantis-mem` in `~/go/bin` / `/usr/local/bin` / `/opt/homebrew/bin`. Check **MCP:
  List Servers → Show Output** for the error.
- **Skill doesn't load** → the `name` in `SKILL.md` frontmatter must be plain kebab-case
  (`mantis-mem`) and match the skill directory name; otherwise it's silently skipped.
- **Reinstall fails / stale state** → delete the cache and retry:
  `~/Library/Application Support/Code/agentPlugins/…` (macOS) ·
  `~/.config/Code/agentPlugins/…` (Linux) · `%APPDATA%\Code\agentPlugins\…` (Windows).

## Caveats

- **Binary discovery + download.** A plugin's `mcp.json` `command` can't be an absolute path,
  so the bundled `./bin/mantis-mem-launcher.js` resolves the binary at runtime — a local
  install (even when VS Code's PATH lacks `~/go/bin`) or, failing that, a checksum-verified
  download of the matching release from GitHub into `~/.mantis/bin`. `MANTIS_NO_DOWNLOAD=1`
  disables the download; `MANTIS_BIN` forces a specific binary.
- **Node dependency.** The launcher itself runs under Node (`command: "node"`) rather
  than as a shell script, since that's the only way to keep a single, unmodified
  `mcp.json` working across macOS, Linux, and Windows.
- **Agent hooks are Preview.** The hook feature (`chat.useCustomAgentHooks`) is marked
  Preview in VS Code, and two specifics are thinly documented — verify them in your build:
  - **Hook location.** VS Code discovers Copilot components under the `com.github.copilot/`
    namespace, so the hooks manifest is at `com.github.copilot/hooks/hooks.json`. If your
    VS Code doesn't pick it up there, try a top-level `hooks/hooks.json` (the docs note
    both forms depending on plugin format).
  - **Command path.** The `bash`/`powershell` commands reference the scripts
    plugin-root-relative (`./com.github.copilot/hooks/mantis-mem-hook.sh`) — the same
    convention `mcp.json` uses successfully here. If the hook can't find the script, its
    command is being resolved from a different base; adjust the relative path accordingly.
  - **Output shape.** Copilot hooks return **JSON** (`hookSpecificOutput.additionalContext`),
    not plain stdout — this differs from Claude Code hooks. The scripts emit that JSON
    (PowerShell via `ConvertTo-Json`; bash via a small dependency-free escaper), and
    `{"continue":true}` as the no-op.

  Because Copilot passes the workspace as `cwd` on stdin, the hook scopes memory to your
  open workspace regardless — no `${workspaceFolder}` variable is needed for the hook.
- **Hook is orientation only (read-only).** It never writes memory, and `UserPromptSubmit`
  is deliberately *not* wired up — no per-prompt capture. Saving stays agent-driven.
- **Project scoping (MCP server).** A plugin runs its server with the **plugin root** as
  the working directory (not your workspace), so the *MCP server* won't auto-scope per
  project under the plugin. (The hook is unaffected — it uses the stdin `cwd`.) For strict
  per-project memory from the server too, register it per workspace in `.vscode/mcp.json`
  instead, launching through the plugin's Node launcher (so the automatic download still
  applies) with `cwd: ${workspaceFolder}`:
  ```json
  {
    "servers": {
      "mantis-mem": {
        "type": "stdio",
        "command": "node",
        "args": [
          "/absolute/path/to/mantis-mem/plugin/vscode/bin/mantis-mem-launcher.js",
          "serve"
        ],
        "cwd": "${workspaceFolder}"
      }
    }
  }
  ```
  You can also pin a name with `MANTIS_PROJECT` in the server's `env`. (To point at a specific
  prebuilt binary instead, set `"command"` to its absolute path with `"args": ["serve"]` — but
  then the auto-download doesn't apply, so the binary must already exist.)

## Recommend the plugin to a repo's users

A repository can suggest/enable this plugin for its collaborators via
`.github/copilot/settings.json` (`enabledPlugins` / `extraKnownMarketplaces`). VS Code
prompts to install recommended plugins on the first chat message in that workspace.
