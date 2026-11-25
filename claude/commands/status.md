---
description: Read-only snapshot of where work stands (done, in progress, next step, blockers)
---

Activate STATUS MODE for: $ARGUMENTS

Read-only. Change nothing. Gather in parallel:
- Engram: `mem_context`, plus `mem_search` for the issue in `$ARGUMENTS` if given.
- `explorer`: current branch, uncommitted changes, ahead/behind vs the remote default branch, `git worktree list`, and active OpenSpec changes: resolve the store per the orchestrator's SDD store resolution (file reads), then list change directories under the store's `openspec/changes/`, excluding `archive/` (skip if no store).
- Linear (`mcp__linear-server`): issues assigned to me In Progress or In Review, plus the issue named in `$ARGUMENTS` or inferred from the branch name.

Reply with three short parts:
1. Done.
2. In progress ("cooking").
3. Next step.

Add blockers if any.
