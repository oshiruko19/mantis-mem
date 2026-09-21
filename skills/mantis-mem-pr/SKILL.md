---
name: mantis-mem-pr
description: >
  Pull request creation workflow for mantis-mem following the issue-first enforcement pattern.
  Trigger: when creating a pull request, opening a pull request, or when preparing changes for review.
metadata:
  author: adel.atrari@gmail.com
  version: "1.0.1"
---

## When to use

Use this skill when:

- Creating a pull request for any change
- Preparing a branch for submission for review
- Helping a contributor open a pull request for their changes

## Critical rules

1. Every PR MUST link a Github issue no exceptions.
2. Every PR MUST have exactly one type:\* label (e.g., type:bug, type:feature, type:refactor).
3. All required automated checks must pass before merge is possible.
4. Blank PRs without issue linkage will be blocked by Github Actions and will not be accepted.

## Workflow

1. Verify issue has `status:approved` label
2. Create branch: feat/_, fix/_, docs/_, refactor/_, chore/\*
3. Implement changes
4. Run tests locally (unit + e2e)
5. Check every changed path against the [Transient Artifact Policy](../../CONTRIBUTING.md#transient-artifact-policy)
6. Open PR using the template
7. Add exactly one type:\* label
8. Wait for all required automated checks to pass
