# Engram Persistent Memory — Protocol

You have access to Engram, a persistent memory system that survives across sessions and compactions.
This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on demand.

---

## Proactive save triggers (mandatory — do NOT wait for user to ask)

Call `mem_save` IMMEDIATELY and WITHOUT BEING ASKED after any of these:

- Architecture or design decision made
- Convention documented or established
- Workflow change agreed upon
- Tool or library choice made with tradeoffs
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- Issue tracker artifact created or updated (Linear/GitHub)
- Configuration change or environment setup done
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned

**Self-check after EVERY task:** "Did I make a decision, fix a bug, learn something non-obvious, or establish a convention? If yes → call `mem_save` NOW."

### Format for `mem_save`

```
title:     Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList")
type:      bugfix | decision | architecture | discovery | pattern | config | preference
scope:     project (default) | personal
topic_key: stable key for evolving topics (e.g. "architecture/auth-model")
content:
  What:    one sentence — what was done
  Why:     what motivated it (user request, bug, performance, etc.)
  Where:   files or paths affected
  Learned: gotchas, edge cases, things that surprised you (omit if none)
```

### Topic update rules

- Different topics MUST NOT overwrite each other
- Same topic evolving → use same `topic_key` (upsert)
- Unsure about key → call `mem_suggest_topic_key` first
- Know exact ID to fix → use `mem_update`

### SDD artifacts

SDD artifacts live only in the OpenSpec filesystem. Do not save, retrieve, or index them in Engram.

---

## When to search memory

### Reactive (user asks)

On any variation of "remember", "recall", "what did we do", "how did we solve",
or references to past work:

1. Call `mem_context` — checks recent session history (fast, cheap)
2. If not found → call `mem_search` with relevant keywords
3. If found → use `mem_get_observation` for full untruncated content

### Proactive (agent decides)

Search PROACTIVELY when:
- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem →
  call `mem_search` with keywords from their message BEFORE responding
- Before any non-SDD phase: check if prior work exists for this change

---

## Session close protocol (mandatory)

Before ending a session or saying "done", call `mem_session_summary`:

```
## Goal
[What we were working on this session]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains to be done — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]
```

This is NOT optional. If you skip this, the next session starts blind.

---

## After compaction

If you see a compaction message or context reset:

1. IMMEDIATELY call `mem_session_summary` with the compacted summary content —
   this persists what was done before compaction
2. Call `mem_context` to recover additional context from previous sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost.

---

## Memory lifecycle

When Engram exposes lifecycle metadata:
- `active` memories → use normally
- `needs_review` memories → stale context, not trusted facts.
  Surface to the user and verify against current evidence before relying on it.
- Do NOT call `mem_review` with action `mark_reviewed` automatically.
  Only after explicit user confirmation.
