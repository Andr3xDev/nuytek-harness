---
description: Safely review or curate Engram project memory
agent: engram-maintainer
context: fork
---

Run `/memory <mode> [argument]` through `engram-maintainer`: $ARGUMENTS

Modes:

1. `review [project]`: read-only assessment. For each stale, redundant, low-value, or recent candidate, explain what it is, why it is a candidate, and the recommended action (keep, update, delete, merge, judge). Never acts: the user decides, then calls `delete` or `merge`.
2. `conflicts [project]`: find pending memory conflicts, explain each pair, and recommend a relation. Records verdicts only for the IDs or pairs the request names; otherwise returns recommendations.
3. `merge <keep-id> <id...>`: fold the other observations into `keep-id`, then delete them. No further confirmation: this request is the approval.
4. `merge-projects <from> <to>`: resolve both projects and counts first, then merge `from` into `to`. No further confirmation: this request is the approval.
5. `delete <observation-id>`: inspect the observation first, then delete only that ID. No further confirmation: this request is the approval.
6. `purge <project>`: resolve scope and count first, then hard-delete without further confirmation. This is irreversible.
7. `export [project]`: state export file and scope first, then create it without further confirmation.
8. `doctor [project]`: read-only health diagnostic.
