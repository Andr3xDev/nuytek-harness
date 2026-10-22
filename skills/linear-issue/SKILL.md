---
name: linear-issue
description: "Trigger: linear issue, linear ticket, linear task, linear project, linear initiative. Generate a Linear issue, project, or initiative body from the base templates."
license: MIT
metadata:
  author: custom
  version: "1.1"
---

## Activation Contract

Use when the user asks to draft or create an issue, a project, or an initiative in Linear.
Read `../md-style-guide/SKILL.md` first; this skill adds only Linear rules.

## Hard Rules

- Every section is optional. Keep only sections with real content; an issue may be just `What to do`.
- No planning content here: an issue states one concrete action, a project states one outcome, an initiative states one long-term direction that groups several projects.
- A diagram, if any, goes inside `Context` or `Notes`.
- Priority, estimate, labels, team, cycle, and project are Linear fields — return them under `metadata`.

## Decision Gates

| **Situation** | **Template** |
|:---|:---:|
| Concrete action, bug, research, or spike | `templates/issue.md` |
| Outcome that groups several issues | `templates/project.md` |
| Long-term direction that groups several projects | `templates/initiative.md` |
| User asks to publish | Show title and body, wait for user OK, then publish via `mcp__linear-server` |
