---
name: context-compact
description: "Trigger: compact, clear, context reset, save state. Save a session summary to Engram only when durable knowledge or pending work exists, before losing context."
license: MIT
metadata:
  author: custom
  version: "1.1"
---

## Activation Contract

Use this skill BEFORE any context loss event: /compact, /clear, session ending,
or when the agent detects context window pressure. Compaction counts as a handoff
(`engram-protocol.md`).

## Hard Rules

- Save before the reset happens — after is too late.
- Save only durable knowledge or pending work. Nothing durable: skip the save.
- Never save SDD artifacts or state to Engram. They live in OpenSpec; the change folder's `state.yaml` carries phase and progress.

## Execution Steps

1. Decide: did the session produce decisions, root-cause fixes, discoveries, preferences, or unfinished work? No: skip to step 3.
2. Call `mem_session_summary` with Goal / Instructions / Discoveries / Accomplished / Next Steps / Relevant Files (file paths, decisions, blockers). For an active SDD change, point to its change folder in Next Steps; do not copy its state.
3. Run the compact/clear. Afterwards call `mem_context` and continue from it.

## Output Contract

Return one line: summary saved (what can be recovered) or skipped (nothing durable). Name any state that could not be preserved (e.g. unsaved file edits).
