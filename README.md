# mantis-mem

A single self-contained binary that gives any MCP-compatible coding agent a
**curated project memory** — not a transcript sink. SQLite + FTS5 full-text
search, exposed through both an **MCP stdio server** and a **human CLI**.

No Node.js, Python, or Docker. One binary, one SQLite file.

```
Agent (Claude Code / Gemini CLI / Codex / VS Code / ...)
    ↓ MCP stdio
mantis-mem (single Go binary)
    ↓
SQLite + FTS5 (~/.mantis/mantis_mem.db)
```

The binary is pure Go — `modernc.org/sqlite` compiles SQLite (with FTS5) into the
program, so there is no cgo and no system SQLite dependency. It cross-compiles to
macOS, Linux, and Windows (amd64/arm64) from a single machine.

## Build

```bash
make build            # -> ./mantis-mem  (CGO_ENABLED=0)
make test             # unit + in-process MCP round-trip tests
make release          # cross-compile the full matrix into dist/
```

Or directly:

```bash
CGO_ENABLED=0 go build -o mantis-mem .
```

## Storage

State lives in one SQLite file, resolved in this order:

1. `--db <path>` flag
2. `$MANTIS_DB`
3. `~/.mantis/mantis_mem.db` (default)

WAL mode is enabled so the MCP server and CLI can touch the file concurrently.

## Project scope

Memory is scoped per project, resolved from the working directory:

1. `$MANTIS_PROJECT` — explicit override (directory-independent)
2. nearest ancestor containing `.git` — the repository root
3. the current working directory

`mem_current_project` (and `mantis-mem project`) report which rule matched.

## MCP tools

| Tool                    | Purpose                                                                             |
| ----------------------- | ----------------------------------------------------------------------------------- |
| `mem_current_project`   | Confirm the resolved project and DB path. Call first to orient.                     |
| `mem_context`           | Recover recent history: latest session summary + recent observation previews.       |
| `mem_search`            | Full-text search the current project. Returns ranked **previews**.                  |
| `mem_timeline`          | Observations in chronological order, optionally scoped to a session/time.           |
| `mem_get_observation`   | Fetch the full record for one observation by id.                                    |
| `mem_save`              | Save durable knowledge; reuse a `topic_key` to update an evolving topic in place.   |
| `mem_session_summary`   | Save/update a session handoff (goal, instructions, discoveries, next steps, files). |
| `mem_suggest_topic_key` | Suggest a stable `namespace/kebab-title` slug.                                      |

Observation kinds: `decision`, `bug`, `discovery`, `config`, `pattern`, `constraint`, `feature`, `note`.

A useful memory is structured:

```
What:    Added retry-safe upload handling.
Why:     Retries could create duplicate records.
Where:   internal/upload/handler.go
Learned: Reuse the request id as the idempotency key.
```

## CLI

```bash
mantis-mem project                       # show resolved project
mantis-mem save --kind bug --title "..." --body "..." [--topic bug/x] [--tags "a b"] [--files "p1,p2"]
mantis-mem search "retry idempotency"    # full-text search
mantis-mem context [--limit N]           # recent history
mantis-mem timeline [--session S] [--since 2026-09-01T00:00:00Z]
mantis-mem get 42                        # full observation
mantis-mem suggest-topic --kind pattern --title "Auth model"
mantis-mem --db /tmp/x.db save ...       # leading global --db works too
```

## Register with an agent

The agent launches `mantis-mem serve` and speaks MCP over stdio.

### Claude Code

```bash
claude mcp add mantis-mem -- /absolute/path/to/mantis-mem serve
```

### Codex, Gemini CLI, and other MCP clients

Add an entry to the client's MCP server config (`command` + `args`). Exact shape
varies by client; most use a `mcpServers` map:

```json
{
  "mcpServers": {
    "mantis-mem": {
      "command": "/absolute/path/to/mantis-mem",
      "args": ["serve"]
    }
  }
}
```

The default DB is `~/.mantis/mantis_mem.db`; override with an absolute path via the
`MANTIS_DB` env var if needed (avoid a literal `~` — it is not expanded).

### VS Code (Copilot, agent mode)

VS Code uses `.vscode/mcp.json` with a `servers` key and `"type": "stdio"`. See
**[plugin/vscode/](plugin/vscode/README.md)** for the template and step-by-step setup + test:

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

`cwd: ${workspaceFolder}` scopes memory to the open VS Code project automatically.

## Operating contract (recommended agent behavior)

1. **Orient** with `mem_current_project`, then `mem_context` / `mem_search`.
2. **Search before repeating** a decision, bug, or convention.
3. **Retrieve progressively**: `mem_search` → `mem_timeline` → `mem_get_observation`.
4. **Save deliberately**: completed fixes, decisions, discoveries, config changes,
   patterns, durable constraints — not raw tool output.
5. **Keep topics stable**: reuse a `topic_key` to update an evolving topic in place.
6. **Leave a handoff** with `mem_session_summary` before ending a session.
