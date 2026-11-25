---
description: Periodic health pass over Engram, CodeGraph and specs; one report, apply only after OK
---

Activate MAINTENANCE MODE for: $ARGUMENTS

Project: `$ARGUMENTS`, default the current project; pass it to each of the three agents. Only orchestrate the agents below; do not redo their logic.

1. In parallel, read-only:
   - `engram-ops`: `/memory` `review` and `doctor` modes for the project.
   - `codegraph-ops`: health check of the index.
   - `specter-ops`: `health` mode (read-only spec health report).
2. Merge into ONE report grouped by area (Engram, CodeGraph, Specs). Each finding gets a recommended action.
3. Apply nothing until the user gives OK.
4. After the OK, route each approved action to its owner agent (Engram actions follow `/memory` mode semantics).
