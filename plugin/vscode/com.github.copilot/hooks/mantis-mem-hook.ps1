#!/usr/bin/env pwsh
# GitHub Copilot agent hook for the VS Code Agent Plugin (Windows) — the
# `powershell` command in hooks.json. Copilot passes session metadata as JSON on
# STDIN (including `cwd` = the workspace folder) and reads a JSON object from
# STDOUT; for SessionStart the hookSpecificOutput.additionalContext string is
# injected into the model's context.
#
# Runs `mantis-mem context` scoped to the workspace and returns the mem_* usage
# contract plus recent memory (ConvertTo-Json handles escaping). Read-only;
# best-effort (emits {"continue":true} and exits 0 on any failure).
$ErrorActionPreference = 'SilentlyContinue'

function Write-Noop { Write-Output '{"continue":true}'; exit 0 }

$stdin = [Console]::In.ReadToEnd()
$event = 'SessionStart'
if ($stdin -match '"hook_event_name"\s*:\s*"([^"]*)"') { $event = $Matches[1] }
$cwd = ''
if ($stdin -match '"cwd"\s*:\s*"([^"]*)"') { $cwd = $Matches[1] }
if (-not $cwd) { $cwd = (Get-Location).Path }

# Locate the mantis-mem binary via the plugin's Node resolver — same discovery as
# the MCP launcher (PATH, install dirs, go env, MANTIS_BIN, and the auto-download
# cache under ~/.mantis/bin). Find-only: the hook never downloads (the launcher
# owns that); on a brand-new install it no-ops until the binary is cached once.
function Find-Bin {
  $root = Join-Path $PSScriptRoot '..\..'
  $bin = (& node (Join-Path $root 'bin\resolve.js') --path 2>$null | Out-String).Trim()
  if ($bin) { return $bin }
  return $null
}

$bin = Find-Bin
if (-not $bin) { Write-Noop }

$context = ''
try {
  Push-Location $cwd
  $context = (& $bin context --limit 5 2>$null | Out-String).Trim()
} catch { }
finally { Pop-Location -ErrorAction SilentlyContinue }
if (-not $context) { Write-Noop }

$header = '[mantis-mem] This project has curated memory via the mem_* MCP tools. Before repeating a decision, bug, or convention, search it (mem_search) and read full records with mem_get_observation. Save durable findings deliberately (mem_save); leave a handoff at the end (mem_session_summary). It is a curated store, not a transcript sink — don''t dump raw output into it. Recent memory:'

$payload = @{
  hookSpecificOutput = @{
    hookEventName    = $event
    additionalContext = "$header`n`n$context"
  }
}
$payload | ConvertTo-Json -Depth 6 -Compress
exit 0
