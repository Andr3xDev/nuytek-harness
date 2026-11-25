---
description: Run every clear-* audit for one scope; one combined report, apply only after OK
---

Activate CLEAR-FULL MODE for: $ARGUMENTS

1. Glob `~/.claude/commands/clear-*.md` at run time; skip `clear-full.md`.
2. Resolve ONE scope up front: `$ARGUMENTS`, else the default-scope rules in `clear-tests.md`. If still unclear, ask the user once before running anything.
3. Run each command's report step with that same explicit scope, so none re-asks or diverges (read-only, may run in parallel). Do not apply anything yet.
4. Present ONE combined report grouped by command.
5. Apply nothing until the user gives OK. After the OK, apply per each command's own apply rules.
