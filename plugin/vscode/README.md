# mantis-mem in VS Code

mantis-mem is an **MCP server**, so "installing it in VS Code" means registering it
with VS Code's built-in MCP support — **not** building a `.vsix` extension. VS Code's
agent mode (Copilot Chat) launches `mantis-mem serve` over stdio and exposes its
`mem_*` tools to the model.

Because mantis-mem resolves the project from its working directory and VS Code sets
the server's `cwd` to the workspace folder, **memory is automatically scoped to the
open VS Code project** (its git root).

> This folder ships a copyable [`mcp.json`](./mcp.json). It is a template — VS Code
> reads `.vscode/mcp.json` in *your* project. (This repo can't commit `.vscode/**`;
> it's excluded by the Transient Artifact Policy.)

## 1. Install the binary

From a clone of this repo (works for the private repo — builds locally, no proxy/auth):

```bash
make install          # local `go install .` -> $(go env GOPATH)/bin/mantis-mem
```

Make sure `~/go/bin` (i.e. `$(go env GOPATH)/bin`) is on your `PATH`, or use the
binary's absolute path in the config below. Confirm:

```bash
mantis-mem version
```

> `go install github.com/oshiruko19/mantis-mem@latest` only works once the repo is
> **public**. While it is private, use `make install` above, or set
> `GOPRIVATE=github.com/oshiruko19/*` and authenticate git to GitHub as an account
> that can read the repo (`gh auth setup-git`).

## 2. Register the server

Pick one:

- **Copy the template.** Create `.vscode/mcp.json` in your project and paste the
  contents of [`mcp.json`](./mcp.json). If `mantis-mem` is not on `PATH`, replace
  `"command": "mantis-mem"` with the full path (e.g. `/Users/you/go/bin/mantis-mem`).
- **Guided flow.** Command Palette → **MCP: Add Server** → **Command (stdio)** →
  command `mantis-mem`, args `serve` → choose **Workspace** (writes `.vscode/mcp.json`)
  or **Global** (user profile, available in every workspace).

Resulting `.vscode/mcp.json`:

```json
{
  "servers": {
    "mantis-mem": {
      "type": "stdio",
      "command": "mantis-mem",
      "args": ["serve"],
      "cwd": "${workspaceFolder}"
    }
  }
}
```

Optional fields: `"env": { "MANTIS_DB": "/abs/path/to.db" }` to use a non-default DB
(the default is `~/.mantis/mantis_mem.db`), or a `"dev": { "watch": "..." }` block to
auto-restart the server.

## 3. Start it and use it

1. Open `.vscode/mcp.json` — a **Start** action appears above the server entry; click
   it. (Or run **MCP: List Servers** → `mantis-mem` → **Start Server**.)
2. Open Chat (`⌃⌘I` / `Ctrl+Alt+I`) and switch the mode selector to **Agent**.
3. Click the **Tools** (🛠) icon in the chat input — you should see mantis-mem's 8
   tools: `mem_current_project`, `mem_context`, `mem_search`, `mem_timeline`,
   `mem_get_observation`, `mem_save`, `mem_session_summary`, `mem_suggest_topic_key`.
4. Ask the agent to use them, approving tool runs when prompted, e.g.:
   > Call `mem_current_project`, then `mem_save` a bug note titled "flaky login test",
   > then `mem_search` for "flaky".

## 4. Verify it works (step by step)

1. **Tools present:** step 3 above shows all 8 `mem_*` tools in the Tools picker.
2. **Correct project:** the `mem_current_project` result's `name` matches your
   workspace folder / git repo, and `source` is `git` (or `cwd`).
3. **Round-trip in chat:** after the agent runs `mem_save` then `mem_search`, the
   search result includes the note you just saved.
4. **Persisted to disk — cross-check from a terminal** in the same project:
   ```bash
   mantis-mem search flaky
   ```
   The CLI (same project scope, same DB) returns the note the agent saved in chat.
5. **Server logs:** **MCP: List Servers** → `mantis-mem` → **Show Output** streams the
   server's stderr — useful if a tool call errors.

## Troubleshooting

- **"command not found" / server won't start** → `mantis-mem` isn't on `PATH`; put the
  absolute path in `"command"`. Check **Show Output** for the reason.
- **Tools don't appear** → confirm the chat is in **Agent** mode, the server shows
  **Running** in **MCP: List Servers**, and you allowed the tools when prompted.
- **Wrong project in `mem_current_project`** → ensure `"cwd": "${workspaceFolder}"` is
  set, or set `MANTIS_PROJECT` via `env` to pin a name.
- **Changes to a freshly rebuilt binary not picked up** → restart the server
  (**MCP: List Servers** → **Restart**), or add a `dev.watch` glob.
