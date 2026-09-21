<!--
  ⚠️ READ BEFORE SUBMITTING

  Every PR must:
  1. Link an approved issue (one with the status:approved label)
  2. Have exactly one type:* label (and only canonical labels)
  3. Pass all required automated checks

  See CONTRIBUTING.md for the full workflow. Do NOT open a PR for an
  unapproved issue — it will be closed.
-->

## 🔗 Linked Issue

<!-- REQUIRED: replace # with the issue number. The issue must have status:approved. -->

Closes #

---

## 🏷️ PR Type

<!-- REQUIRED: check exactly ONE, then add the matching type:* label to the PR. -->

- [ ] `type:bug` — Bug fix
- [ ] `type:feature` — New feature
- [ ] `type:question` — Question requiring tracked work
- [ ] `type:docs` — Documentation only
- [ ] `type:refactor` — Refactoring (no behavior change)
- [ ] `type:chore` — Maintenance, dependencies, tooling
- [ ] `type:breaking-change` — Breaking change (major version bump)

---

## 📝 Summary

<!-- What does this PR do? 1–3 bullets. -->

-

## 📂 Changes

| File           | Change       |
| -------------- | ------------ |
| `path/to/file` | What changed |

## 🧪 Test Plan

<!-- How did you verify this? -->

- [ ] Unit tests pass locally: `go test ./...`
- [ ] `gofmt`/`vet` clean: `make fmt && make vet`
- [ ] Manually tested the affected CLI command / MCP tool
- [ ] _(when applicable)_ E2E tests pass: `go test -tags e2e ./internal/mcpserver/...` — 🔜 suite not present yet
- [ ] _(when applicable)_ Lint passes: `make lint` — 🔜 target not present yet

---

## 🤖 Automated Checks

These run on every PR. Required checks must pass before merge.

| Check                               | Verifies                                         | Status today  |
| ----------------------------------- | ------------------------------------------------ | ------------- |
| **Unit Tests**                      | `go test ./...`                                  | ✅ Active     |
| **gofmt / vet / build / cross**     | formatting, `go vet`, build, cross-compile matrix| ✅ Active     |
| **Check Issue Reference**           | PR body has `Closes #N` / `Fixes #N` / `Resolves #N` | 🔜 Planned |
| **Check Issue Has status:approved** | the linked issue is approved                     | 🔜 Planned    |
| **Check PR Has type:\* Label**      | exactly one canonical `type:*` label             | 🔜 Planned    |

---

## ✅ Contributor Checklist

- [ ] I linked an **approved** issue above (`Closes #N`)
- [ ] I added exactly **one** `type:*` label to this PR
- [ ] I ran unit tests locally: `go test ./...`
- [ ] I ran `make fmt && make vet`
- [ ] Docs updated (if behavior changed)
- [ ] Commits follow [conventional commits](https://www.conventionalcommits.org/)
- [ ] **No `Co-Authored-By` trailers** in commits
- [ ] No paths forbidden by the Transient Artifact Policy (binary, `dist/`, `*.db*`, …)

## 💬 Notes for Reviewers

<!-- Optional: context, tradeoffs, open questions. -->
