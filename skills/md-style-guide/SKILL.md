---
name: md-style-guide
description: "Trigger: producing Markdown, document, report, README, planning file, SDD artifact, issue, PR, or release body. Apply personal Markdown conventions to any artifact, plus the extra rules for tracker bodies."
license: MIT
metadata:
  author: custom
  version: "2.1"
---

# MD Style Guide

Personal Markdown conventions for any written artifact. Tracker bodies (issues, PRs, releases) add the rules in `Tracker bodies`.
These are preferences, not a fixed layout: when the user wants a specific format, they ask the agent directly.

## Conventions

- **Bold** only when the term has real relevance in context; never for whole sentences or as a habit.
- *Italics* sparingly, for emphasis on a single word.
- Inline `code` for technical terms, commands, file paths, and values; not for emphasis.
- Use `-` for bullets, never `*`.
- Nest bullets only when the content has a real hierarchy.
- Prefer prose over bullets when it reads better.
- Ordered lists only when sequence or priority matters.
- No emojis.
- Readability over density; consistency over local cleverness.
- When in doubt: less formatting, more clarity.
- Standard GitHub-flavored Markdown.
- Never invent facts: links, IDs, estimates, owners, versions, or test results come from the user or a tool.
- Delete a section with nothing real to say. Never write "N/A" or filler.
- Checklists use `- [ ]`; each item is observable and verifiable.
- Neutral, professional tone.

## Mermaid

Use a diagram only when it explains faster than prose. One diagram per section; size is whatever the idea needs.

| **Need** | **Diagram** |
|:---|:---:|
| Flow, steps, decisions | `flowchart LR` |
| Interaction between services or actors | `sequenceDiagram` |
| States of an entity | `stateDiagram-v2` |
| Phases or milestones in time | `gantt` |
| Goal broken into epics/tasks | `flowchart TD` |

- Node labels are short nouns or verbs; quote labels that contain `()` or `:`.
- No styling, no themes, no click handlers.
- If a diagram only repeats a short list, drop the diagram.

## Tracker bodies

Only for issues, PRs, and releases written through the tracker skills. The template is a base guide, not a fixed form: keep its order as the default, and add, rename, or split sections when the user asks or the context needs it.

- The title goes in the tool's title field, never as `#` in the body. Bodies start at `##`.
- No HTML.
- Remove every `<!-- ... -->` hint and `[placeholder]` before output.
- `---` only where the template already has it; delete it with an emptied section.
- Reference issues and PRs by ID only (`#123`, `owner/repo#123`, `ENG-123`), never by branch name, and only when needed.
- Templates are in English; headings and content use the session language, default English.

### Execution Steps

1. Pick the variant from the skill's decision gates and load that file from `templates/`.
2. Gather inputs from the conversation, diff, spec, or tracker MCP; ask only for what cannot be derived.
3. Fill the template, then apply these rules and the skill's own rules.
4. Return title and body separately. Publish only after showing the draft and getting explicit user OK.

### Output Contract

```
title: <one line>
body: <filled markdown>
metadata: <tool fields that are not markdown (labels, estimate, assignee) or none>
open_questions: <missing required info, or none>
```

## Control Points

- If the document type is ambiguous, confirm before structuring.
- Do not apply this style retroactively to existing files without explicit instruction.
