---
description: Standards review of a scope (default is uncommitted changes plus branch changes vs the default branch)
---

Activate REVIEW MODE for: $ARGUMENTS (default scope below when empty)

Default scope when empty:
- Git repo: uncommitted changes plus branch changes vs the repo's default branch. If the base is unclear, ask the user; never assume `main`.
- No git: files touched in this session plus files related to the current task.
- Still undeterminable: ask the user for the scope.

1. Delegate to `code-reviewer` for a full review: all lenses, against the owner quality standards.
2. Present a severity-ranked report.

For an adversarial dual review, run /judgment-day (skill).
