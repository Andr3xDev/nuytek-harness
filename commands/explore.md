---
description: Quick read-only pre-check of a codebase question before any SDD, fix, or other work
---

Activate EXPLORE MODE for: $ARGUMENTS

1. Detect working directory and project context.
2. Delegate to `explorer` (not sdd-explore) with one specific question from "$ARGUMENTS". If several repos are mentioned, include all their paths.
3. Present the answer with `path:line` evidence.
4. End with a suggested next step (SDD, fix, or nothing). Never start it.

Read-only: no files modified, no specs created.
