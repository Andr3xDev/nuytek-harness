---
description: Audit tests for overlap, low value, staleness, and coverage gaps; report first, apply only after OK
---

Activate CLEAR-TESTS MODE for: $ARGUMENTS

Default scope when empty, limited to the tests touching it:
- Git repo: uncommitted changes plus branch changes vs the repo's default branch. If the base is unclear, ask the user; never assume `main`.
- No git: files touched in this session plus files related to the current task.
- Still undeterminable: ask the user for the scope.

Flag:
- Overlap and duplication, stale tests, and disproportionate tests.
- Low-value tests: mappers, DTOs, models without logic, getters.
- Real coverage gaps in business logic, branches, validation, permissions, money, security, parsers, transformations, or regressions.

1. Delegate to `code-reviewer` in focused mode; pass the Flag list above as the checklist.
2. REPORT FIRST: `file:line`, the issue, and the proposed action (remove, merge, rewrite, add).
3. Apply nothing until the user gives OK. Every test removal needs explicit approval.
4. After the OK, delegate to `builder` (or `test-writer` for approved new tests).
