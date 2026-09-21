#!/usr/bin/env bash
# Configure branch protection on `main` so the issue-first workflow is enforced:
# require the PR Policy checks + CI + one approving review before merge.
#
# ⚠️ Requires ADMIN on the repo. This changes live repo settings — review before running.
#
# Usage (from anywhere in the repo):
#   ./scripts/setup-branch-protection.sh
#   GH_REPO=owner/name BRANCH=main ./scripts/setup-branch-protection.sh
#
# Requires: gh (authenticated as an admin) and jq.
set -euo pipefail

command -v gh >/dev/null 2>&1 || { echo "error: gh (GitHub CLI) not found" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "error: jq not found" >&2; exit 1; }

: "${GH_REPO:=$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
: "${BRANCH:=main}"

# Status check contexts that must pass. These are the job *names*:
#   - the three jobs in .github/workflows/pr-policy.yml
#   - the CI job in .github/workflows/ci.yml
# If you rename a job, update the matching name here.
REQUIRED_CHECKS=(
  "Check Issue Reference"
  "Check Issue Has status:approved"
  "Check PR Has type:* Label"
  "fmt · vet · test · build"
  "E2E Tests"
  "lint (golangci-lint)"
)

echo "Repo:   $GH_REPO"
echo "Branch: $BRANCH"
printf 'Required checks:\n'; printf '  - %s\n' "${REQUIRED_CHECKS[@]}"

contexts_json="$(printf '%s\n' "${REQUIRED_CHECKS[@]}" | jq -R . | jq -s .)"

payload="$(jq -n --argjson contexts "$contexts_json" '{
  required_status_checks: { strict: true, contexts: $contexts },
  enforce_admins: false,
  required_pull_request_reviews: {
    required_approving_review_count: 1,
    dismiss_stale_reviews: true
  },
  restrictions: null,
  required_linear_history: true,
  allow_force_pushes: false,
  allow_deletions: false
}')"

echo "Applying protection to $GH_REPO@$BRANCH ..."
printf '%s' "$payload" | gh api -X PUT "repos/$GH_REPO/branches/$BRANCH/protection" \
  -H "Accept: application/vnd.github+json" --input -

echo
echo "Done. Tip: also restrict merges to squash-only under Settings → General → Pull Requests,"
echo "or run: gh api -X PATCH repos/$GH_REPO -f allow_squash_merge=true -F allow_merge_commit=false -F allow_rebase_merge=false"
