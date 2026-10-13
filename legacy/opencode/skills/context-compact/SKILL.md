---
name: context-compact
description: "Trigger: compact, clear, context reset, save state. Preserve active session state to Engram before losing context."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use this skill BEFORE any context loss event: /compact, /clear, session ending,
or when the agent detects context window pressure. Automatically triggered by
the pre-compact hook if configured.

## Hard Rules

- Always persist to Engram before the context reset happens — after is too late.
- Save the full state, not just a summary — include file paths, decisions, blockers.
- If an SDD change is in progress, save the DAG state with current phase and progress.
- Never skip this because "there's nothing important" — let Engram decide relevance via decay.

## Decision Gates

| Situation | Action |
|---|---|
| SDD change active | Save DAG state + current phase + task progress |
| Debug in progress | Save root cause analysis so far + files investigated |
| General work | Save decisions made, files modified, current intent |
| Nothing meaningful happened | Save a minimal session note — still not optional |

## Execution Steps

1. Identify what's active: SDD change? Debug? General task?
2. Collect state:
   - Current intent (what the user asked for)
   - Decisions made this session
   - Files read or modified
   - Blockers or open questions
   - If SDD: current phase, task progress, artifact keys written
3. Call `mem_save`:
   ```
   title: "Session state before compact"
   topic_key: "session/pre-compact"
   type: "architecture"
   content: {collected state as structured markdown}
   ```
4. If SDD change active, also save DAG state:
   ```
   title: "sdd/{change-name}/state"
   topic_key: "sdd/{change-name}/state"
   type: "architecture"
   content: {phase, progress, pending tasks}
   ```
5. Confirm to the user that state was preserved before proceeding with compact/clear.

## Output Contract

Return confirmation with:
- What was saved (topic keys)
- What can be recovered next session
- Any state that could NOT be preserved (e.g. unsaved file edits)
