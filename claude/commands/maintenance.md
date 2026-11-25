---
description: Periodic health pass over Engram, CodeGraph and specs; one report, apply only after OK
---

Activate MAINTENANCE MODE for: $ARGUMENTS

Project: `$ARGUMENTS`, default the current project. Only orchestrate the agents below; do not redo their logic.
Inputs: repo path from the current repo (or `~/dev/specter/repos.yaml`); store id via the orchestrator's SDD store resolution, matching `stores.<id>.engram` to the project.

1. In parallel, read-only:
   - `engram-ops`: `/memory` `review` and `doctor` modes, given the Engram project name.
   - `codegraph-ops`: health check of the index, given the repo path (current repo, or the project's repo).
   - `specter-ops`: `health [store-id]` (read-only spec health report), with the store id resolved per the orchestrator's SDD store resolution; none checks all stores.
2. Merge into ONE report grouped by area (Engram, CodeGraph, Specs). Each finding gets a recommended action.
3. Apply nothing until the user gives OK.
4. After the OK, route only the actions and IDs the user approved to their owner agent (Engram actions follow `/memory` mode semantics):
   - CodeGraph `sync`/`index`: delegation carries explicit AUTH for that action.
   - Spec findings that need spec content edits: propose an SDD change. `specter-ops` applies only fixes within its editable files (`repos.yaml`, references, READMEs, pointers).
