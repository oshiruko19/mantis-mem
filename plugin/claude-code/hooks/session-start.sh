#!/usr/bin/env bash
# Claude Code SessionStart / post-compaction hook (macOS / Linux / Git Bash).
#
# Claude Code injects this script's STDOUT into the model's context. It prints the
# mem_* usage contract plus this project's recent memory (`mantis-mem context`) —
# the deterministic counterpart to the advisory skill. Read-only: it never writes
# memory (saving stays deliberate via mem_save / mem_session_summary). Best-effort:
# any failure prints nothing and exits 0, so a session is never blocked.
set -u

# Consume the hook payload on stdin (JSON with cwd/session_id/...). We read it so
# the writer never gets SIGPIPE, and use `cwd` if present.
INPUT="$(cat 2>/dev/null || true)"

# Project dir precedence: stdin `cwd` -> $CLAUDE_PROJECT_DIR -> current dir.
# mantis-mem resolves the project from this dir (nearest ancestor with .git).
project_dir() {
  local d
  d="$(printf '%s' "$INPUT" \
    | sed -n 's/.*"cwd"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$d" ] || d="${CLAUDE_PROJECT_DIR:-}"
  [ -n "$d" ] || d="$PWD"
  printf '%s' "$d"
}

# Locate the mantis-mem binary via the plugin's Node resolver — same discovery as
# the MCP launcher (PATH, install dirs, go env, MANTIS_BIN, and the auto-download
# cache under ~/.mantis/bin). Find-only: the hook never triggers a download (the
# launcher owns that, so we stay within the hook timeout). On a brand-new install
# the hook simply no-ops until the launcher has cached the binary once.
find_bin() {
  local root="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
  node "$root/bin/resolve.js" --path 2>/dev/null
}

BIN="$(find_bin)"; [ -n "$BIN" ] || exit 0   # not resolvable — the MCP server surfaces that
DIR="$(project_dir)"

CONTEXT="$(cd "$DIR" 2>/dev/null && "$BIN" context --limit 5 2>/dev/null)" || exit 0
[ -n "$CONTEXT" ] || exit 0

cat <<EOF
[mantis-mem] This project has curated memory via the mem_* MCP tools. Before repeating a
decision, bug, or convention, search it (mem_search) and read full records with
mem_get_observation. Save durable findings deliberately (mem_save); leave a handoff at the
end (mem_session_summary). It is a curated store, not a transcript sink — don't dump raw
output into it. Recent memory:

$CONTEXT
EOF
exit 0
