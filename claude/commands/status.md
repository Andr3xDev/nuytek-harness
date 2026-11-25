---
description: Read-only snapshot of where work stands (done, in progress, next step, blockers)
---

Activate STATUS MODE for: $ARGUMENTS

Read-only. Change nothing. Gather in parallel:
- Engram: `mem_context`, plus `mem_search` for the issue in `$ARGUMENTS` if given.
- `explorer`: current branch, uncommitted changes, ahead/behind vs the branch's upstream (from `git status -sb`; refs are local and may be stale), and `git worktree list`.
- Active OpenSpec changes: the orchestrator resolves the store itself (Read, per its SDD store resolution), then Globs the change directories under `<store-root>/openspec/changes/`, excluding `archive/` (skip if no store).
- Linear (`mcp__linear-server`): issues assigned to me In Progress or In Review, plus the issue named in `$ARGUMENTS` or inferred from the branch name.

Reply with three short parts:
1. Done.
2. In progress ("cooking").
3. Next step.

Add blockers if any.
