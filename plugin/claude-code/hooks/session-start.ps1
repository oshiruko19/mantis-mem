#!/usr/bin/env pwsh
# Claude Code SessionStart / post-compaction hook (Windows PowerShell).
#
# PowerShell counterpart of session-start.sh, for Windows without Git Bash. Wire it
# up by setting the hook's "shell" to "powershell" and pointing "command" at this
# file (see hooks.json / the plugin README). Claude Code injects STDOUT into the
# model's context: the mem_* usage contract plus this project's recent memory.
# Read-only; best-effort (prints nothing and exits 0 on any failure).
$ErrorActionPreference = 'SilentlyContinue'

# Consume the hook payload on stdin (JSON) and pull out `cwd` if present.
$stdin = [Console]::In.ReadToEnd()
$cwd = ''
if ($stdin -match '"cwd"\s*:\s*"([^"]*)"') { $cwd = $Matches[1] }
if (-not $cwd) { $cwd = $env:CLAUDE_PROJECT_DIR }
if (-not $cwd) { $cwd = (Get-Location).Path }

# Locate the mantis-mem binary via the plugin's Node resolver — same discovery as
# the MCP launcher (PATH, install dirs, go env, MANTIS_BIN, and the auto-download
# cache under ~/.mantis/bin). Find-only: the hook never downloads (the launcher
# owns that); on a brand-new install it no-ops until the binary is cached once.
function Find-Bin {
  $root = if ($env:CLAUDE_PLUGIN_ROOT) { $env:CLAUDE_PLUGIN_ROOT } else { Join-Path $PSScriptRoot '..' }
  $bin = (& node (Join-Path $root 'bin\resolve.js') --path 2>$null | Out-String).Trim()
  if ($bin) { return $bin }
  return $null
}

$bin = Find-Bin
if (-not $bin) { exit 0 }   # not resolvable — the MCP server surfaces that itself

$context = ''
try {
  Push-Location $cwd
  $context = (& $bin context --limit 5 2>$null | Out-String).Trim()
} catch { }
finally { Pop-Location -ErrorAction SilentlyContinue }
if (-not $context) { exit 0 }

$header = @'
[mantis-mem] This project has curated memory via the mem_* MCP tools. Before repeating a
decision, bug, or convention, search it (mem_search) and read full records with
mem_get_observation. Save durable findings deliberately (mem_save); leave a handoff at the
end (mem_session_summary). It is a curated store, not a transcript sink — don't dump raw
output into it. Recent memory:
'@

Write-Output "$header`n`n$context"
exit 0
