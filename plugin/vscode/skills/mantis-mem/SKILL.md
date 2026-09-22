---
name: mantis-mem
description: >
  Use the mantis-mem MCP tools (mem_*) to read and write curated project memory.
  Trigger whenever starting work in a repository that has mantis-mem installed,
  before revisiting a past decision/bug/convention, after completing a fix or
  decision worth remembering, or when ending a session (handoff).
---

# mantis-mem — curated project memory

This repository uses **mantis-mem** for durable, curated project memory exposed as
MCP tools (`mem_*`). Treat it as project knowledge, **not** a transcript sink.

## Critical rules

1. **Orient before writing.** Call `mem_current_project` to confirm the resolved
   project, then `mem_context` and `mem_search` to recover relevant history before
   starting related work.
2. **Search before repeating.** Before revisiting a decision, bug, convention, or
   request that may already be known, `mem_search` with focused terms. Results are
   previews, not the complete record.
3. **Retrieve progressively.** `mem_search` for candidates → `mem_timeline` when
   surrounding session context matters → `mem_get_observation` before relying on a
   full observation.
4. **Save deliberately.** Save completed bug fixes, decisions, discoveries,
   configuration changes, patterns, and durable user constraints with `mem_save`.
   Do not capture raw tool output or every conversational turn.
5. **Capture progress mid-task.** For longer work, `mem_save` the record once, then
   use `mem_append` to add timestamped progress notes to it as you go, instead of
   waiting until the end or overwriting the record. Appends preserve prior content
   and stay searchable.
6. **Keep evolving knowledge stable.** Give an evolving topic a stable `topic_key`
   such as `architecture/auth-model` and reuse it to update that topic instead of
   creating competing memories. `mem_save` returns near-duplicate detection nudges
   if similar entries exist. Use `mem_suggest_topic_key` when unsure.
7. **Leave and review handoffs.** Before ending a session, save a `mem_session_summary` with
   the goal, instructions, discoveries, accomplished work, next steps, and files. Call
   `mem_session_history` when onboarding to see past session summaries.

## Cookbook

- **If** you're starting a task in this repo → **then** call `mem_current_project`,
  then `mem_search` the task's key terms or `mem_session_history` for recent handoffs.
  _Example:_ working on auth → `mem_search "auth session cookie"`.
- **If** you just fixed a bug or made a decision → **then** `mem_save` it.
  _Example:_ `mem_save(kind="bug", title="Retry-safe upload", topic_key="bug/upload-dupes",
body="What: reuse request id as idempotency key. Why: retries duplicated rows.
Where: internal/upload/handler.go")`.
- **If** you're mid-task on something already saved → **then** `mem_append` a progress
  note to it instead of re-saving or overwriting.
  _Example:_ `mem_append(id=42, note="Progress: wired the store method; tests green.")`.
- **If** a topic already exists (search or `mem_save` duplicate nudge found it) → **then**
  reuse its `topic_key` in `mem_save` to update it in place instead of creating a duplicate.
- **If** you're wrapping up → **then** `mem_session_summary(...)` with next steps.

A useful memory is structured:

```
What:    Added retry-safe upload handling.
Why:     Retries could create duplicate records.
Where:   internal/upload/handler.go
Learned: Reuse the request id as the idempotency key.
```
