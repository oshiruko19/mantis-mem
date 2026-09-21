#!/usr/bin/env bash
# GitHub Copilot agent hook for the VS Code Agent Plugin (macOS / Linux) — the
# `bash` command in hooks.json. Copilot passes session metadata as JSON on STDIN
# (including `cwd` = the workspace folder) and reads a JSON object from STDOUT; for
# SessionStart the `hookSpecificOutput.additionalContext` string is injected into
# the model's context.
#
# This runs `mantis-mem context` scoped to the workspace and returns the mem_* usage
# contract plus recent memory — the dynamic counterpart to the always-on
# .github/copilot-instructions.md. Read-only; best-effort (emits {"continue":true}
# and exits 0 on any failure, so a session is never blocked).
set -u

INPUT="$(cat 2>/dev/null || true)"

# no-op response: valid JSON that changes nothing.
noop() { printf '%s' '{"continue":true}'; exit 0; }

# Extract a top-level string field from the stdin JSON without needing jq.
json_field() {
  local key="$1"
  [[ "$INPUT" =~ \"$key\"[[:space:]]*:[[:space:]]*\"([^\"]*)\" ]] && printf '%s' "${BASH_REMATCH[1]}"
}

# Escape a string for embedding in a JSON double-quoted value (bash-native).
json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

# Locate the mantis-mem binary via the plugin's Node resolver — same discovery as
# the MCP launcher (PATH, install dirs, go env, MANTIS_BIN, and the auto-download
# cache under ~/.mantis/bin). Find-only: the hook never downloads (the launcher
# owns that); on a brand-new install it no-ops until the binary is cached once.
find_bin() {
  local root; root="$(cd "$(dirname "$0")/../.." && pwd)"
  node "$root/bin/resolve.js" --path 2>/dev/null
}

EVENT="$(json_field hook_event_name)"; [ -n "$EVENT" ] || EVENT="SessionStart"
DIR="$(json_field cwd)"; [ -n "$DIR" ] || DIR="$PWD"

BIN="$(find_bin)"; [ -n "$BIN" ] || noop   # not resolvable — the MCP server surfaces that

CONTEXT="$(cd "$DIR" 2>/dev/null && "$BIN" context --limit 5 2>/dev/null)" || noop
[ -n "$CONTEXT" ] || noop

HEADER="[mantis-mem] This project has curated memory via the mem_* MCP tools. Before repeating a decision, bug, or convention, search it (mem_search) and read full records with mem_get_observation. Save durable findings deliberately (mem_save); leave a handoff at the end (mem_session_summary). It is a curated store, not a transcript sink — don't dump raw output into it. Recent memory:"

ESCAPED="$(json_escape "$HEADER"$'\n\n'"$CONTEXT")"
printf '{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"%s"}}' "$EVENT" "$ESCAPED"
exit 0
