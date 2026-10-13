---
name: zenhub-issue
description: "Trigger: zenhub, zenhub epic, zenhub goal, zenhub task, planning issue. Generate a planning issue for a ZenHub epic, goal, or task."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use when the user asks to plan or summarize an epic, goal, or task tracked in ZenHub.
Read `../md-style-guide/SKILL.md` first; this skill adds only ZenHub rules.

## Hierarchy

An epic has N goals; a goal has N tasks. Each issue describes only its own level.

## Hard Rules

- These issues are plans: they explain what will be done, not a bug or a request.
- `Plan` is an explanation first: what will be done, in what order, and why. A Mermaid diagram may follow to support it, never to replace it. Drop the diagram when it would only repeat a list.
- `Scope` makes the final objective clear: the end state, what is in, and what is out.
- Parent/child relations are managed in the ZenHub app.
- ZenHub issues are GitHub issues: the body must render on GitHub, Mermaid included.
- Estimate, pipeline, sprint, and relations are ZenHub fields — return them under `metadata`.

## Decision Gates

| **Situation** | **Template** |
|:---|:---:|
| Large deliverable with user-facing value | `templates/epic.md` |
| Outcome inside an epic | `templates/goal.md` |
| One unit of work inside a goal | `templates/task.md` |
| User asks to publish | If the ZenHub MCP (`mcp__zenhub__*`) is connected: show draft, wait for user OK, publish. Otherwise return the draft only |
