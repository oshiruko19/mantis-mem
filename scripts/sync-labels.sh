#!/usr/bin/env bash
# Create or update the canonical mantis-mem labels in the GitHub repo from
# .github/labels.yml, using the GitHub CLI. Idempotent (uses --force to upsert).
#
# Usage (from anywhere in the repo):
#   ./scripts/sync-labels.sh                 # sync to the repo of the current dir
#   GH_REPO=owner/name ./scripts/sync-labels.sh
#
# Requires: gh (authenticated).  It does NOT delete labels not in the manifest —
# remove those manually after an audit.
set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"
LABELS_FILE=".github/labels.yml"

command -v gh >/dev/null 2>&1 || { echo "error: gh (GitHub CLI) not found" >&2; exit 1; }
[ -f "$LABELS_FILE" ] || { echo "error: $LABELS_FILE not found" >&2; exit 1; }

# Parse the flat YAML into name<TAB>color<TAB>description triples. Each label is a
# name/color/description block in that order; we emit on the description line.
awk '
  /^- name:/                  { line=$0; sub(/^[^:]*: */,"",line); gsub(/^"|"$/,"",line); name=line }
  /^[[:space:]]+color:/       { line=$0; sub(/^[^:]*: */,"",line); gsub(/^"|"$/,"",line); color=line }
  /^[[:space:]]+description:/  { line=$0; sub(/^[^:]*: */,"",line); gsub(/^"|"$/,"",line);
                                printf "%s\t%s\t%s\n", name, color, line }
' "$LABELS_FILE" | while IFS=$'\t' read -r name color desc; do
  [ -n "$name" ] || continue
  printf 'syncing %-26s #%s\n' "$name" "$color"
  gh label create "$name" --color "$color" --description "$desc" --force
done

echo "labels synced."
