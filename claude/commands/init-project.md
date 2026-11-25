---
description: Detect and set up Engram, CodeGraph and OpenSpec store for a repo; create only what is missing after OK
---

Activate INIT-PROJECT MODE for: $ARGUMENTS

1. Detect, read-only (`explorer`):
   - `.engram/config.json` at the repo root (Engram project name).
   - `.codegraph/` index.
   - The repo's OpenSpec store: resolve it per the orchestrator's SDD store resolution.
2. Report what exists and what is missing, with proposed values. Project name default: `$ARGUMENTS`, else the repo name. Store: existing or new.
3. Wait for the user's OK. Store creation or registration and CodeGraph indexing need it.
4. Create only the missing pieces:
   - `.engram/config.json` and store/`repos.yaml`: `specter-ops`.
   - CodeGraph index: `codegraph-ops`.
5. Report the result. `~/dev/specter` is not committed unless the user asks.
