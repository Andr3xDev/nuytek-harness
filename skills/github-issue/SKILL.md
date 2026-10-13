---
name: github-issue
description: "Trigger: github issue, bug report, feature request, open an issue. Generate a standard GitHub issue (bug or feature) for repos without a planning tracker."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use for genuine issues in GitHub-only repos without a planning structure: personal, freelance, or unstructured projects. Not for work owned by a Linear team (use `linear-issue`).
Read `../md-style-guide/SKILL.md` first; this skill adds only GitHub rules.

## Hard Rules

- No planning content: one problem or one request per issue.
- If the repo has `.github/ISSUE_TEMPLATE/`, its template wins.
- Labels, assignees, and milestone are GitHub fields — return them under `metadata`.
- Logs go in fenced blocks, trimmed to the relevant lines.

## Decision Gates

| **Situation** | **Template** |
|:---|:---:|
| Something is broken | `templates/bug.md` |
| New capability or change | `templates/feature.md` |
| Plan for a goal, epic, or task in ZenHub | Use `zenhub-issue` |
| User asks to publish | Show title and body, wait for user OK, then `mcp__github__create_issue` |
