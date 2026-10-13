---
name: release-notes
description: "Trigger: release notes, GitHub release, changelog entry, release message. Draft release notes from a general structure shaped with the user per context."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use when the user asks for release notes, a GitHub release body, or a release message for chat.
Read `../md-style-guide/SKILL.md` first; this skill adds only release rules.

## Hard Rules

- The template is a general base. Detail level, subsections, and technical depth are agreed with the user for each release.
- Labels (change categories) are defined by the user. Never invent or default them; ask if not given.
- Before drafting, confirm only what is missing:
  1. Range (previous tag → target ref) or the list of changes
  2. Channel: chat message or GitHub release
  3. Audience and technical level
  4. Labels to group changes under
- Source of truth is the merged PRs or commits in that range; read PR descriptions when commits are unclear.
- Never list every commit or fill sections by default — include only what the user chose.
- Draft only: the user publishes. Never run `gh release create` or any release tool.
- Breaking changes always appear, with the migration step.

## Decision Gates

| **Situation** | **Action** |
|:---|:---|
| Range unknown | Ask — never guess the previous tag |
| Chat message | Standard Markdown, no platform-specific syntax: no `##` headings or tables; bold lead lines and `-` bullets, same order |
| GitHub release | Full template `templates/release.md`; if `CHANGELOG.md` exists, follow its format |
