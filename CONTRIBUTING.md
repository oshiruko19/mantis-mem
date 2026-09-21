# Contributing to mantis-mem

Thanks for contributing. mantis-mem aims for a strict **issue-first workflow** — every change starts with an approved issue.

| Area                                                                                     | Status today                                                           |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Unit tests — `go test ./...`                                                             | ✅ Active (CI: `.github/workflows/ci.yml`)                             |
| `gofmt` · `go vet` · build · cross-compile matrix                                        | ✅ Active (CI)                                                         |
| PR template                                                                              | ✅ Active (`.github/PULL_REQUEST_TEMPLATE.md`)                         |
| Repo skills (`skills/`) + PR skill                                                       | ✅ Active                                                              |
| Make targets: `build test e2e vet lint fmt tidy install clean release`                   | ✅ Active (`Makefile`)                                                 |
| Issue forms + canonical labels                                                           | ✅ Active (`.github/ISSUE_TEMPLATE/`, `.github/labels.yml`)            |
| Issue-first gating checks (Issue Reference / status:approved / type:\* label)            | ⚙️ Workflow present (`pr-policy.yml`); enforced after `scripts/setup-branch-protection.sh` |
| Label sync (`scripts/sync-labels.sh`)                                                    | ⚙️ Script present; run once with `gh` to create labels                 |
| E2E tests (`go test -tags e2e ./internal/mcpserver/...`)                                 | ✅ Active (`make e2e`, CI job "E2E Tests")                             |
| Lint — `make lint` (golangci-lint v2.13.2: errcheck/staticcheck/unused)                  | ✅ Active (`.golangci.yml`, CI job "lint (golangci-lint)")             |
| Stale bot (30d → `status:stale`, weekly sweep)                                           | ✅ Active (`.github/workflows/stale.yml`)                              |
| `make deadcode-check` · `make perf-check` ratchets                                       | 🔜 Planned (targets not in `Makefile`)                                 |
| Label migration policy (`label-policy.mjs`), merge queue                                 | 🔜 Planned                                                             |
| Transient-artifact validator                                                             | 🔜 Planned (policy documented below; `.gitignore` covers common cases) |
| `setup.sh` agent-skill linking                                                           | 🔜 Planned                                                             |
| Plugin / npm hygiene                                                                     | ➖ N/A (no `plugin/` package yet)                                      |

---

## Local Development

```bash
make build     # -> ./mantis-mem   (CGO_ENABLED=0, pure Go, no cgo)
make test      # go test ./...
make fmt       # gofmt -w .
make vet       # go vet ./...
make release   # cross-compile the OS/arch matrix into dist/
```

Install the binary onto your `PATH`:

```bash
make install    # local `go install .` -> $(go env GOPATH)/bin/mantis-mem
```

`go install github.com/oshiruko19/mantis-mem@latest` works only once the repo is
**public**. While it is private, use `make install`, or set
`GOPRIVATE=github.com/oshiruko19/*` with git authenticated to an account that can
read the repo.

---

## Contribution Workflow

```
Open Issue → Get status:approved → Open PR → Add type:* label → Review & Merge
```

### Step 1: Open an Issue

Use the correct template:

- **Bug Report** — for bugs
- **Feature Request** — for new features or improvements
- **Documentation Improvement** — for missing, outdated, or unclear docs
- **Tracked Question** — for questions requiring maintainer investigation, a repository change, or a durable decision

> ⚠️ Blank issues should be disabled once templates land. General questions and
> support belong in [Discussions](https://github.com/oshiruko19/mantis-mem/discussions).

Fill in all required fields. Issues should receive the `status:needs-review` label.

If useful for alignment, search existing issues before opening a new one.

> _Active:_ four issue forms in `.github/ISSUE_TEMPLATE/` each auto-apply a `type:*`
> label plus `status:needs-review`. One-time setup: a maintainer runs
> `./scripts/sync-labels.sh` (needs `gh`) so those labels exist in the repo.

### Step 2: Wait for Approval

A maintainer reviews the issue and replaces `status:needs-review` with `status:approved` if it's accepted for implementation.

**Do not open a PR until the issue is approved.**

> _🔜 Planned:_ automated checks that block PRs referencing unapproved issues are not yet active (see the status table). Until then this step is enforced by review.

### Step 3: Open a Pull Request

Once the issue is approved:

1. Fork the repo and create a branch from `main` (`feat/…`, `fix/…`, `docs/…`, `refactor/…`, `chore/…`)
2. Implement your change
3. Open a PR using the PR template — **link the approved issue** with `Closes #N`
4. Add exactly **one `type:*` label** to the PR (see label system below)

### Step 4: Automated PR Checks

Target required contexts on every PR (and merge-queue group, once enabled):

| Check                               | What it verifies                                                 | Status                                   |
| ----------------------------------- | ---------------------------------------------------------------- | ---------------------------------------- |
| **Unit Tests**                      | `go test ./...` — all tests except those tagged `//go:build e2e` | ✅ Active                                |
| **Check Issue Reference**           | PR body contains `Closes #N`, `Fixes #N`, or `Resolves #N`       | ⚙️ In `pr-policy.yml`; required after branch-protection setup |
| **Check Issue Has status:approved** | The linked issue has the `status:approved` label                 | ⚙️ In `pr-policy.yml`; required after branch-protection setup |
| **Check PR Has type:\* Label**      | Exactly one canonical `type:*` label                             | ⚙️ In `pr-policy.yml`; required after branch-protection setup |
| **E2E Tests**                       | `go test -tags e2e ./internal/mcpserver/...`                     | ✅ Active                                |
| **Lint**                            | golangci-lint v2.13.2 (`make lint`)                              | ✅ Active                                |

CI (`.github/workflows/ci.yml`) runs on every PR: `gofmt`/`go vet`/`go test ./...`/build,
the cross-compile matrix, **E2E Tests**, and **lint (golangci-lint)**. The three
issue-first gates live in `.github/workflows/pr-policy.yml` and run on every PR too, but
only **block merges** once a maintainer makes them required — run
`scripts/setup-branch-protection.sh` once (needs `gh` + admin) to require all of them.

---

## Transient Artifact Policy

PR validation should inspect the complete changed-file set. Deleted artifacts are
allowed; enforcement applies to added, modified, copied, and renamed destination
paths. The following transient artifacts are rejected unless a path is explicitly
described as repository-root-only:

| Enforced class                  | Forbidden paths and variants                                                                          |
| ------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Agent-tool state                | Any directory named `.atl` (at any depth) and `**/mantis-dev/**`                                      |
| Generated agent links           | Repository-root `.claude/skills/**`, `.codex/skills/**`, `.github/skills/**`, and `.gemini/skills/**` |
| Transient development documents | Repository-root `plan.md`, `agent-report.md`, `agent-handoff.md`, and `handoff.md`                    |
| Local data                      | `**/*.db`, `**/*.db-wal`, `**/*.db-shm`, and `**/mantis-mem-export.json`                              |
| Binaries                        | Repository-root `mantis-mem`, `dist/**` build outputs, and `**/*.exe`                                 |
| OS metadata                     | `**/.DS_Store` and `**/Thumbs.db`                                                                     |
| Editor metadata                 | `**/.idea/**` and `**/.vscode/**`                                                                     |
| Editor and backup files         | `**/*.swp`, `**/*.swo`, and `**/*~`                                                                   |

Canonical, reviewable documentation is allowed — e.g. `docs/plan.md` and
`specs/transient-artifact-policy.md` are documentation, not root transient
development documents. The transient document names above are forbidden only at the
repository root; do not use documentation paths to retain ephemeral local notes.

> _🔜 Planned:_ a CI validator that enforces this on the changed-file set. Today the
> `.gitignore` already excludes the binary, `dist/`, and `*.db*` files; the rest is
> enforced by review.

### Quality Ratchets — 🔜 Planned

The intended gate: every PR and push to `main` runs `make deadcode-check`, which
analyzes all module packages with `golang.org/x/tools/cmd/deadcode` and compares
stable `file<TAB>symbol` identities against `.deadcode-baseline.txt`. New unreachable
functions fail CI; removed entries tighten the baseline (refresh it deliberately with
`make deadcode-baseline` in the same change). _Not yet wired: the `deadcode-check`/
`deadcode-baseline` targets and the baseline file do not exist yet._

### Performance Ratchet — 🔜 Planned

The intended gate: pushes to `main` compare the store search/scan benchmarks against
the previous `main` SHA on the same runner, failing on significant slowdowns. Locally,
`make perf-check` runs against a committed, host-specific baseline; refresh with
`make perf-baseline` and justify the change. _Not yet wired: no benchmarks, targets,
or baseline exist yet._

### Lint — ✅ Active

CI runs golangci-lint **v2.13.2** with `errcheck`, `staticcheck`, and `unused`
(config in `.golangci.yml`). Run `make lint` before pushing — the target requires
that exact version on `PATH` and prints an install hint otherwise:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
```

_Refinement still planned:_ the "only new findings" ratchet (report only issues the
PR introduces). Today the check runs against the whole tree, which is currently clean.

---

## Label System

> _🔜 Planned:_ these labels and the automation that enforces their cardinality
> (`label-policy.mjs`, stale bot) are the target vocabulary; they are not yet created
> on the repo. Define/apply them manually until the automation lands.

### Type Labels (required on every PR — pick exactly one)

| Label                  | Use for                                        |
| ---------------------- | ---------------------------------------------- |
| `type:bug`             | Bug fixes                                      |
| `type:feature`         | New features                                   |
| `type:question`        | Questions requiring tracked work               |
| `type:docs`            | Documentation-only changes                     |
| `type:refactor`        | Code refactoring with no behavior change       |
| `type:chore`           | Maintenance, tooling, dependencies             |
| `type:breaking-change` | Breaking changes (requires major version bump) |

### Status Labels (set by maintainers)

| Label                       | Meaning                                                                        |
| --------------------------- | ------------------------------------------------------------------------------ |
| `status:needs-review`       | Awaiting maintainer review (auto-applied to new issues)                        |
| `status:approved`           | Approved for implementation — PRs can now be opened                            |
| `status:in-progress`        | Actively being worked on — auto-exempt from stale bot                          |
| `status:blocked`            | Blocked by another issue or external dependency                                |
| `status:stale`              | No activity for 30 days — auto-applied by stale bot                            |
| `status:wontfix`            | Closed without implementation — applied to stale, rejected, or duplicate items |
| `status:possible-duplicate` | Potential duplicate under evaluation                                           |

### Resolution Labels (set after closure)

| Label                  | Meaning                           |
| ---------------------- | --------------------------------- |
| `resolution:duplicate` | Confirmed duplicate after closure |

Replace the current `status:*` label with `status:possible-duplicate` while evaluating a report. After confirming and closing the duplicate, replace it with `status:wontfix` and apply `resolution:duplicate`.

### Priority Labels (set by maintainers)

`priority:critical`, `priority:high`, `priority:medium`, `priority:low`

> Issues with `priority:critical`, `priority:high`, and `status:approved` should never be auto-closed by the stale bot.

### Effort Labels (set by maintainers, for contributor guidance)

| Label           | Meaning                                             |
| --------------- | --------------------------------------------------- |
| `effort:small`  | < 1 hour — good starting point for new contributors |
| `effort:medium` | 1–4 hours                                           |
| `effort:large`  | > 4 hours or spans multiple files                   |

### Namespace Contract

| Namespace      | Issues           | Pull requests  |
| -------------- | ---------------- | -------------- |
| `type:*`       | Exactly one      | Exactly one    |
| `status:*`     | Exactly one      | At most one    |
| `priority:*`   | At most one      | Not applicable |
| `resolution:*` | At most one      | Not applicable |
| `effort:*`     | Multiple allowed | Not applicable |
| `size:*`       | Not applicable   | At most one    |

The only unnamespaced labels are maintainer-owned protected exceptions: `good first issue` and `help wanted`.

---

## PR Rules

- Keep PR scope focused — one logical change per PR.
- Use [conventional commits](https://www.conventionalcommits.org/) format.
- Ensure local checks pass before pushing:
  - Unit: `go test ./...` — ✅ available
  - Format/vet: `make fmt && make vet` — ✅ available
  - E2E: `make e2e` (`go test -tags e2e ./internal/mcpserver/...`) — ✅ available
  - Lint: `make lint` (golangci-lint v2.13.2) — ✅ available
- Update docs in the same PR when behavior changes.
- Do not reference endpoints/scripts that do not exist in code.
- **Do not include `Co-Authored-By` trailers in commits.**
- Do not include paths prohibited by the [Transient Artifact Policy](#transient-artifact-policy).

### Conventional Commit Format

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

**Examples (scopes match this repo's packages):**

```
feat(cli): add --json flag to search output

fix(store): prevent duplicate observation on topic upsert

docs(contributing): document label system and transient-artifact policy

refactor(mcpserver): extract tool registration into a table

chore(deps): bump modernc.org/sqlite

fix!: change session_id storage format (breaking change)
BREAKING CHANGE: session ids are now UUIDs instead of integers
```

Types map to labels: `feat` → `type:feature`, `fix` → `type:bug`, `docs` → `type:docs`, `refactor` → `type:refactor`, `chore` → `type:chore`.

---

## Skill Authoring Standard

Repository skills live in `skills/` (e.g. `skills/mantis-mem-pr/SKILL.md`), indexed by `AGENTS.md`.

Use a **hybrid format**:

1. Structured base (purpose, when to use, critical rules, checklists)
2. Cookbook section (`If / Then / Example`) for repetitive actions

Why hybrid:

- Structured base protects correctness and architecture intent
- Cookbook improves execution consistency for common flows

---

## Maintainer Triage Cadence

| Activity         | Frequency     | What Happens                                    |
| ---------------- | ------------- | ----------------------------------------------- |
| New issue triage | Within 2 days | Maintainer labels + approves or closes          |
| PR review        | Within 7 days | Maintainer reviews + requests changes or merges |
| Backlog sweep    | Weekly        | Approved/blocked issues reassessed              |
| Label audit      | Monthly       | Orphan labels removed; accuracy check           |

If you haven't received a response within 7 days on a PR or issue, a single ping comment is welcome.

---

## What Gets Closed Without Merging

- PRs opened without an approved issue
- PRs that fail CI and aren't updated within 30 days
- Issues that are vague, a duplicate, or belong in [Discussions](https://github.com/oshiruko19/mantis-mem/discussions)
- Issues with no response to a maintainer question after 14 days

---

## Agent Skill Linking — 🔜 Planned

The intended helper links repo `skills/*` into project-local agent directories
(`.claude/skills/*`, `.codex/skills/*`, `.gemini/skills/*`) so each agent picks them
up. _Not yet wired: `setup.sh` does not exist yet; these generated link directories
are covered by the Transient Artifact Policy and must not be committed._
