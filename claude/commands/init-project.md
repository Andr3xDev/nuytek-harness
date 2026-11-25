---
description: Detect and set up Engram, CodeGraph and OpenSpec store for a repo; create only what is missing after OK
---

Activate INIT-PROJECT MODE for: $ARGUMENTS

1. Detect, read-only:
   - The orchestrator resolves the repo's OpenSpec store itself (Read, per its SDD store resolution).
   - `explorer` checks `.engram/config.json` (Engram project name) and the `.codegraph/` index.
2. Report what exists and what is missing, with proposed values. Project name default: `$ARGUMENTS`, else the repo name. Store: existing or new.
3. Wait for the user's OK. Store creation or registration and CodeGraph indexing need it.
4. Create only the missing pieces:
   - Store missing or repo not in `repos.yaml`: `specter-ops` `register <repo-path> <store-id>` (sets up the store only if it does not exist).
   - `.engram/config.json` or `openspec/config.yaml` pointer: `specter-ops` `pointer <repo-path> <store-id>`. Give the project name only when `.engram/config.json` is missing or wrong; it must match the `stores.<id>.engram` entry for the repo, so `register` runs first. If only `openspec/config.yaml` is missing, omit the project name.
   - Engram project unknown to Engram: `engram-ops` project creation.
   - CodeGraph index: `codegraph-ops` `codegraph init <repo>`, with explicit AUTH.
5. Report the result. `~/dev/specter` is not committed unless the user asks.
