---
description: Review comments and docs for noise and staleness; report first, apply only after OK
---

Activate CLEAR-COMMENTS MODE for: $ARGUMENTS

Covers code comments, docstrings, and repo docs.

Default scope when empty:
- Git repo: uncommitted changes plus branch changes vs the repo's default branch. If the base is unclear, ask the user; never assume `main`.
- No git: files touched in this session plus files related to the current task.
- Still undeterminable: ask the user for the scope.

Rule: keep only comments that carry a non-obvious why, constraint, risk, workaround, or externally imposed behavior. Flag comments that narrate code, restate names, or are stale. Flag docs that are redundant or outdated.

1. Delegate to `code-reviewer` in focused mode; pass the Rule list above as the checklist.
2. REPORT FIRST: `file:line`, the issue, and the proposed action (remove, rewrite, keep).
3. Apply nothing until the user gives OK.
4. After the OK, delegate to `quick-builder` (trivial) or `builder`.
