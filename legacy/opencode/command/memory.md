---
description: Safely review or curate Engram project memory
agent: engram-maintainer
---

Run `/memory <mode> [argument]` through `engram-maintainer`: $ARGUMENTS

Modes:

1. `review [project]`: read-only assessment. Return stale, redundant, low-value, and recent candidate IDs with reasons and recommended actions. Never delete automatically.
2. `delete <observation-id>`: inspect the observation first, then delete only that ID after native confirmation.
3. `purge <project>`: show resolved scope and count first, then hard-delete only after native confirmation. This is irreversible.
4. `export [project]`: state export file and scope first, then create it only after native confirmation.
5. `doctor [project]`: read-only health diagnostic.
