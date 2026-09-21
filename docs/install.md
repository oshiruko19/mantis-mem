# Installing mantis-mem

mantis-mem is a single self-contained Go binary that gives an MCP-compatible coding agent a
**curated project memory** (SQLite + FTS5), exposed over an MCP stdio server. This guide
covers installing it as a **Claude Code plugin** or a **VS Code Agent Plugin** (for GitHub
Copilot). For other clients (Codex, Gemini CLI, raw `claude mcp add`) see the top-level
[README](../README.md).

**You don't build anything.** The plugin ships a small launcher that, on first use,
downloads the prebuilt `mantis-mem` binary for your OS/arch from this repo's
[GitHub Releases](https://github.com/oshiruko19/mantis-mem/releases) (checksum-verified) and
caches it under `~/.mantis/bin`. No `make`, no Go toolchain. Building from source is optional
and covered at the end.

- **Claude Code**
  - [Install the Claude Code plugin](#install-the-claude-code-plugin)
  - [Verify](#verify)
  - [How to use in claude-code](#how-to-use-in-claude-code)
- **VS Code (GitHub Copilot)**
  - [Install the VS Code Agent Plugin](#install-the-vs-code-agent-plugin)
  - [Verify (VS Code)](#verify-vs-code)
  - [How to use in VS Code](#how-to-use-in-vs-code)
- [How the binary is obtained](#how-the-binary-is-obtained)
- [Build from source (contributors)](#build-from-source-contributors)
- [Troubleshooting](#troubleshooting)

> **Node.js** must be available — Claude Code already requires it. The plugin starts the
> server (and performs the one-time binary download) through a small Node launcher, so a
> single command works on macOS, Linux, and Windows without a per-OS override.

---

## Install the Claude Code plugin

This repository doubles as a Claude Code **plugin marketplace** — the marketplace manifest
at [`.claude-plugin/marketplace.json`](../.claude-plugin/marketplace.json) points at the
plugin in [`plugin/claude-code/`](../plugin/claude-code/).

### Option A — from GitHub (recommended)

In Claude Code, add the repo as a marketplace and install the plugin:

```
/plugin marketplace add oshiruko19/mantis-mem
/plugin install mantis-mem@mantis-mem
```

The name is `mantis-mem@mantis-mem` — `plugin-name@marketplace-name` — because the plugin
and the marketplace share the name. Restart Claude Code (or reload) if prompted.

The **first** time the server starts, the launcher downloads the matching binary from GitHub
Releases (a few seconds, once) and caches it at `~/.mantis/bin/mantis-mem-<version>`. Later
sessions reuse the cached binary.

### Option B — from a local checkout

If you already have the repo cloned, point the marketplace at the local path instead of
GitHub — handy while developing the plugin:

```
/plugin marketplace add /absolute/path/to/mantis-mem
/plugin install mantis-mem@mantis-mem
```

### Option C — enable automatically for a project (settings.json)

To have the plugin enabled for everyone who trusts a project, add this to the project's
`.claude/settings.json` (checked into the repo). Claude Code registers the marketplace and
enables the plugin when the user trusts the folder — no slash commands needed:

```json
{
  "extraKnownMarketplaces": {
    "mantis-mem": {
      "source": { "source": "github", "repo": "oshiruko19/mantis-mem" }
    }
  },
  "enabledPlugins": {
    "mantis-mem@mantis-mem": true
  }
}
```

---

## Verify

1. **Server connected.** Run `/mcp` — `mantis-mem` should be listed as connected, exposing
   the `mem_*` tools. (On a brand-new install, give the first-run download a moment.)
2. **Binary cached.** A `mantis-mem-<version>` file should now exist under `~/.mantis/bin`
   (unless you already had one on PATH / in `~/go/bin`, which is used as-is).
3. **Right project.** Ask Claude to call `mem_current_project` (or run `mantis-mem project`
   in a terminal at the repo root). `source: git` with your repo's name and path confirms
   memory is scoped to this project, not the plugin directory.
4. **Skill loaded.** Run `/plugin` and confirm `mantis-mem` is enabled; its `mantis-mem`
   skill ships with it.

---

## How to use in claude-code

Once installed, the plugin gives Claude Code three things: the **`mem_*` MCP tools**, a
**skill** that tells Claude when to reach for them, and a **`SessionStart` hook** that
orients Claude deterministically at the start of every session. You mostly don't call
anything by hand — Claude uses the tools as it works — but here's the model.

### What you get

| Tool                    | Use it to…                                                                    |
| ----------------------- | ----------------------------------------------------------------------------- |
| `mem_current_project`   | Confirm the resolved project and DB path. Orient first.                       |
| `mem_context`           | Recover recent history: latest session summary + recent observation previews. |
| `mem_search`            | Full-text search the project's memory. Returns ranked **previews**.           |
| `mem_timeline`          | Observations in chronological order (optionally scoped to a session/time).    |
| `mem_get_observation`   | Fetch the full record for one observation (with commit SHA + staleness).      |
| `mem_save`              | Save durable knowledge; reuse a `topic_key` to update a topic in place.       |
| `mem_session_summary`   | Save a session handoff (goal, discoveries, next steps, files).                |
| `mem_session_history`   | List past session summaries / handoffs, newest first.                         |
| `mem_suggest_topic_key` | Suggest a stable `namespace/kebab-title` slug.                                |

### The operating contract (what the skill nudges Claude to do)

1. **Orient** with `mem_current_project`, then `mem_context` / `mem_search` before starting
   related work.
2. **Search before repeating** a decision, bug, or convention — results are previews, not
   the whole record.
3. **Retrieve progressively**: `mem_search` → `mem_timeline` → `mem_get_observation`.
4. **Save deliberately** — completed fixes, decisions, discoveries, config changes,
   patterns, durable constraints. Not raw tool output.
5. **Keep topics stable** — reuse a `topic_key` (e.g. `architecture/auth-model`) to update
   an evolving topic instead of creating competing entries.
6. **Leave a handoff** with `mem_session_summary` before ending a session.

A memory reads best when it's structured:

```
What:    Added retry-safe upload handling.
Why:     Retries could create duplicate records.
Where:   internal/upload/handler.go
Learned: Reuse the request id as the idempotency key.
```

### Two layers: the skill (advisory) and the hook (deterministic)

The contract above is delivered two ways, on purpose:

- The **skill** teaches Claude *when* to use the tools, but triggers softly — the model may
  not always invoke it.
- The **`SessionStart` hook** guarantees orientation: it runs `mantis-mem context` and
  injects the usage contract plus this project's recent memory into **every** session, and
  again **after compaction** — so memory survives context loss (the same behavior other
  memory plugins lean on hooks for).

The hook is **read-only** — it never writes to memory. mantis-mem is a **curated** store,
*not* a transcript sink, so the plugin deliberately does **not** auto-capture your prompts
or transcript on every turn. Saving stays deliberate, driven by the agent through `mem_save`
and `mem_session_summary`. (If you specifically want passive capture, it can be added as an
opt-in hook, but it works against the curated model.)

### In practice

You can just work — Claude will search memory before repeating itself and save durable
findings as it goes. To steer it explicitly, ask in plain language, for example:

- *"Check mantis-mem for anything we decided about the auth session model."* → `mem_search`
- *"What's the recent history on this repo?"* → `mem_context` / `mem_session_history`
- *"Save this fix to memory under topic `bug/upload-dupes`."* → `mem_save`
- *"Leave a handoff for next session with the next steps."* → `mem_session_summary`

### Per-project scoping

Memory is kept **separately per project**. Claude Code sets `CLAUDE_PROJECT_DIR` to the
project root, and the plugin runs the server there, so mantis-mem resolves the project from
the nearest ancestor containing `.git`. Open a different repo and you get that repo's
memory. Confirm anytime with `mem_current_project`.

All state lives in one SQLite file — `~/.mantis/mantis_mem.db` by default. Override with an
absolute path via the `MANTIS_DB` environment variable (avoid a literal `~`; it isn't
expanded).

---

## Install the VS Code Agent Plugin

For **GitHub Copilot** in VS Code, mantis-mem ships as a [VS Code Agent
Plugin](https://code.visualstudio.com/docs/agent-customization/agent-plugins) in
[`plugin/vscode/`](../plugin/vscode/). Enabling it bundles the MCP server (`mem_*` tools),
the usage skill, and a Copilot `SessionStart` / `PreCompact` **hook** that injects the
usage contract and this workspace's recent memory at the start of every session. As with
Claude Code, the binary is downloaded automatically on first use — nothing to build.

### Enable the required settings

In **Settings (JSON)**:

- **`chat.plugins.enabled`** → `true` — agent plugins are experimental and off by default.
- **`chat.useCustomAgentHooks`** → `true` — Preview; needed for the orientation hook. Without
  it the MCP server and skill still work, you just lose the deterministic session-start
  memory injection.

### Register the plugin folder

Point the **`chat.pluginLocations`** setting at the plugin directory (workspace
`.vscode/settings.json` for one project, or user settings to enable it everywhere):

```json
{
  "chat.plugins.enabled": true,
  "chat.useCustomAgentHooks": true,
  "chat.pluginLocations": {
    "/absolute/path/to/mantis-mem/plugin/vscode": true
  }
}
```

It then appears under **Extensions view → “Agent Plugins – Installed”**, where you can
enable/disable it globally or per workspace (or from the Chat view: gear icon → **Plugins**).

> **Install from source (git URL)** — VS Code's *Chat: Install Plugin From Source* expects
> `plugin.json` at the **repo root**. This plugin lives in a subdirectory (`plugin/vscode/`),
> so use the local-folder method above.

### Per-project memory scoping (optional but recommended)

A plugin runs its MCP **server** from the plugin root, not your workspace, so memory won't
auto-scope per project from the plugin alone. For strict per-project memory, register the
server per workspace in `.vscode/mcp.json`, launching it **through the plugin's Node
launcher** so you still get the automatic binary download *and* a workspace-scoped `cwd`:

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

`cwd: ${workspaceFolder}` scopes the server to the open project. (The **hook** is unaffected —
it scopes via the `cwd` Copilot passes on stdin.) You can also pin a fixed name with
`MANTIS_PROJECT` in the server's `env`. If you'd rather point at a specific prebuilt binary
instead of the launcher, set `"command"` to that binary's absolute path and `"args": ["serve"]`
— but then the automatic download doesn't apply, so the binary must already exist.

---

## Verify (VS Code)

1. **MCP: List Servers** (Command Palette) → `mantis-mem` is listed and **Running**. (Allow a
   moment for the first-run download.)
2. Chat (`⌃⌘I`) → **Agent** mode → **Configure Tools** (🛠): the `mem_*` tools appear.
3. **Chat: Configure Skills** → the `mantis-mem` skill is listed.
4. Ask the agent to *"`mem_save` a note then `mem_search` it"* and confirm the round-trip.
5. **Hook** (needs `chat.useCustomAgentHooks`): start a fresh chat session in a repo that has
   memory — the agent should open already aware of recent memory (latest session summary +
   recent observations) without being asked. Sanity-check the hook script from a terminal:
   ```bash
   printf '{"hook_event_name":"SessionStart","cwd":"'"$PWD"'"}' \
     | bash plugin/vscode/com.github.copilot/hooks/mantis-mem-hook.sh
   ```
   It should print JSON with a populated `hookSpecificOutput.additionalContext` (or
   `{"continue":true}` if there's no memory yet, or the binary isn't cached yet).

---

## How to use in VS Code

The `mem_*` tools and the operating contract are **identical to Claude Code** — see
[What you get](#what-you-get) and [the operating contract](#the-operating-contract-what-the-skill-nudges-claude-to-do)
above. Copilot uses the tools as it works; steer it in plain language (*"check mantis-mem
for what we decided about auth"*, *"save this fix to memory"*).

What differs is **how** VS Code is oriented to use them — three layers, weakest to strongest:

1. **Skill** (`skills/mantis-mem/SKILL.md`) — advisory; loads only when its description
   matches the task.
2. **Repo instructions** (`.github/copilot-instructions.md`) — *always* applied to every
   request, but only **static** text (the contract; it can't run anything). This is an
   optional repo file, not part of the plugin.
3. **Copilot hook** (`com.github.copilot/hooks/hooks.json`) — the deterministic, **dynamic**
   layer: it runs `mantis-mem context` and injects the contract **plus this workspace's
   recent memory** into every session (and before compaction). This is the VS Code
   equivalent of the Claude Code `SessionStart` hook.

Like the Claude Code hook, the Copilot hook is **read-only** — it never writes memory. There
is no passive/auto capture on every prompt; saving stays deliberate via `mem_save` /
`mem_session_summary`. Memory is scoped per workspace (the hook uses the workspace `cwd`;
scope the server too with the `.vscode/mcp.json` above). State lives in one SQLite file
(`~/.mantis/mantis_mem.db` by default; override with an absolute `MANTIS_DB`).

---

## How the binary is obtained

On startup the launcher resolves the `mantis-mem` binary in this order, and only downloads if
nothing is already available:

1. **`MANTIS_BIN`** — if set to an existing file, that binary is used (bring-your-own).
2. **A local install** — `mantis-mem` on `PATH`, in `~/go/bin`, `/usr/local/bin`,
   `/opt/homebrew/bin`, or via `go env GOBIN`/`GOPATH`. A `make install` / `go build` binary
   wins here, so contributors always run their own build.
3. **The cached download** — `~/.mantis/bin/mantis-mem-<version>` from a previous run.
4. **Download** — fetch `mantis-mem_<version>_<os>_<arch>.(tar.gz|zip)` from the matching
   GitHub Release, verify its **sha256** against the release `checksums.txt`, extract, and
   cache it under `~/.mantis/bin`. The version is pinned to the plugin version, so a plugin
   update pulls a matching binary.

**Environment overrides:**

| Variable             | Effect                                                                          |
| -------------------- | ------------------------------------------------------------------------------- |
| `MANTIS_BIN`         | Absolute path to a binary to use instead of downloading (dev builds, mirrors).  |
| `MANTIS_NO_DOWNLOAD` | Set to `1` to disable auto-download entirely (air-gapped / policy).             |
| `MANTIS_VERSION`     | Pin/override the binary version to fetch (defaults to the plugin version).      |
| `MANTIS_REPO`        | Override the source repo `owner/name` (forks / internal mirrors).               |

The hooks locate the binary through the **same** resolver but never trigger a download (the
MCP launcher owns that, so hooks stay within their timeout). On a brand-new install the very
first session's orientation hook may no-op until the launcher has cached the binary once; it
self-heals from the next session.

---

## Build from source (contributors)

Building is only needed to hack on mantis-mem itself, or where auto-download isn't wanted.
It's pure Go (`CGO_ENABLED=0`, no system SQLite):

```bash
make install        # builds and installs to ~/go/bin/mantis-mem (step 2 above finds it)
# or:
go install github.com/oshiruko19/mantis-mem@latest   # pulls from GitHub and builds
# or:
CGO_ENABLED=0 go build -o mantis-mem .                # then point MANTIS_BIN at it
```

Because a local install is preferred over the download (resolution step 2), any of these is
picked up automatically. To force the plugin to use a specific build regardless of location,
set `MANTIS_BIN=/absolute/path/to/mantis-mem`.

---

## Troubleshooting

### Claude Code

- **`/mcp` shows mantis-mem failed / "could not obtain the binary".** The first-run download
  couldn't complete. Check connectivity to `github.com`; behind a proxy, set the usual
  `HTTPS_PROXY`. As a fallback, download the release for your OS/arch from
  [Releases](https://github.com/oshiruko19/mantis-mem/releases) and either put it on `PATH`
  or set `MANTIS_BIN` to it. Air-gapped? set `MANTIS_NO_DOWNLOAD=1` and provide `MANTIS_BIN`.
- **"checksum mismatch".** A corrupted/interrupted download or a proxy rewriting the asset.
  Delete `~/.mantis/bin` and retry, or install a binary manually and point `MANTIS_BIN` at it.
- **`node: command not found`.** Node isn't on the PATH Claude Code launched with. Install
  Node (or ensure your version manager's shims are on PATH) and restart.
- **Memory is scoped to the wrong place** (e.g. `source: cwd` and a path that isn't your
  repo). Confirm the folder is a git repo (`git status`) — scoping anchors on the nearest
  `.git`. As a directory-independent override, set the `MANTIS_PROJECT` environment
  variable to a fixed project name.
- **Plugin not listed after install.** Run `/plugin marketplace update`, then `/plugin` to
  re-enable it. Validate the marketplace manifest with `claude plugin validate .` from the
  repo root.

### VS Code (Copilot)

- **Plugin doesn't appear** → confirm `chat.plugins.enabled` is `true`, and that the path in
  `chat.pluginLocations` points at the folder containing `plugin.json` (`plugin/vscode`).
- **MCP server won't start / no `mem_*` tools** → either Node isn't on PATH (the launcher and
  the binary download both run under `node`), or the download failed. Check **MCP: List
  Servers → Show Output** for the error; then retry connectivity, or set `MANTIS_BIN` to a
  manually downloaded binary.
- **Hook doesn't fire** → confirm `chat.useCustomAgentHooks` is `true` (Preview). It's a thinly
  documented feature: if the hook is still ignored, the plugin's hook path
  (`com.github.copilot/hooks/hooks.json`) or the script command's resolution base may differ
  in your build — see the [plugin README](../plugin/vscode/README.md#caveats).
- **Skill doesn't load** → the `name` in `SKILL.md` frontmatter must be plain kebab-case
  (`mantis-mem`) and match the skill directory name, or it's silently skipped.
- **Reinstall fails / stale state** → delete the cache and retry:
  `~/Library/Application Support/Code/agentPlugins/…` (macOS) ·
  `~/.config/Code/agentPlugins/…` (Linux) · `%APPDATA%\Code\agentPlugins\…` (Windows).
