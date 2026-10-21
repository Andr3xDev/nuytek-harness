---
description: Safely review or curate Engram project memory
agent: engram-maintainer
context: fork
---

Run `/memory <mode> [argument]` through `engram-maintainer`: $ARGUMENTS

Modes:

1. `review [project]`: read-only assessment. Return stale, redundant, low-value, and recent candidate IDs with reasons and recommended actions. Never delete automatically.
2. `delete <observation-id>`: inspect the observation first, then delete only that ID. No further confirmation: this request is the approval.
3. `purge <project>`: resolve scope and count first, then hard-delete without further confirmation. This is irreversible.
4. `export [project]`: state export file and scope first, then create it without further confirmation.
5. `doctor [project]`: read-only health diagnostic.
