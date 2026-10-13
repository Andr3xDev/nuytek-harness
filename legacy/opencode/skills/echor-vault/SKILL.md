---
name: echor-vault
description: "Trigger: echor vault, read echor, update echor, onboard project to echor, echor frontmatter, echor consult, echor validate. Shared contract for reading and writing the Echor Obsidian vault."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use this skill whenever an agent reads or writes the Echor vault at
`~/dev/echor`. Echor centralizes knowledge about the user's repos/projects.
Echor never duplicates content from Specter (`~/dev/specter/openspec`) or
Linear — it only references them by ID or path.

## 1. Paths and Naming

Vault root: `~/dev/echor`.

```
~/dev/echor/
  README.md
  _templates/project.md
  _templates/component.md
  projects/<slug>/index.md
  projects/<slug>/decisions.md
  projects/<slug>/components/<component>.md   (only when the component warrants its own note)
```

Slug = the repo's directory name, lowercase-kebab.

A component gets its own note under `components/` ONLY when it has its own
stack or its own relations. Otherwise it is only listed in the parent's
`components` frontmatter field — no note is created for it.

## 2. Frontmatter Schema

```yaml
type: service              # service | personal-project | umbrella | component (required)
status: active              # active | paused | archived (required)
repo: https://github.com/... | local path | null   # required; umbrella uses null
openspec_id: null          # id/path under ~/dev/specter/openspec, or null (required, nullable)
linear_project_id: null    # required, nullable
stack: [python, fastapi, dynamodb]   # required, may be empty list
relates_to: ["[[projects/lucyna/index|lucyna]]"] # path wikilinks (required, may be empty list)
components: [ingest-worker]  # required, may be empty list
part_of: "[[projects/lucyna/index|lucyna]]" # only on type: component; otherwise null
last_indexed_commit: a1b2c3d  # required
last_indexed_at: 2026-09-14   # required
---
```

Rules:

- `relates_to` entries MUST be quoted path wikilinks in the form
  `"[[projects/<slug>/index|<slug>]]"`. A plain string does not create a graph
  edge and is invalid.
- `part_of` is set only on notes with `type: component`. On any other type it
  stays `null`; its value uses the matching quoted path wikilink.
- Every project has `index.md` and `decisions.md`. `index.md` is the overview
  and links `[[decisions]]`. `decisions.md` contains only closed decisions that
  affect future work.
- `type: umbrella` has `repo: null`.
- `last_indexed_commit` and `last_indexed_at` track the repo HEAD at the time
  of the last vault write. This pair is the Updater's change signal — if the
  repo's current HEAD differs from `last_indexed_commit`, the note is stale.
- Below the frontmatter is free prose for the human: what the project is, why
  it exists, how it fits with the rest of the portfolio.

## 3. Read Recipes

Deterministic operations, no custom parser required:

- List all projects: `ls ~/dev/echor/projects`
- Read one project: Read `~/dev/echor/projects/<slug>/index.md`
- Read closed decisions: Read `~/dev/echor/projects/<slug>/decisions.md`
- Who relates to X: `rg -l '"\[\[projects/X/index\|X\]\]"' ~/dev/echor/projects`
- Who uses a stack entry: `rg -l "fastapi" ~/dev/echor/projects`
- List components of a project: read its `components` field, then
  `ls ~/dev/echor/projects/<slug>/components`

## 4. Write Rules

- Edit files in place. Never write a parallel draft file.
- Preserve the human prose below the frontmatter untouched unless it is
  factually wrong.
- Never reorder frontmatter keys.
- Never delete a key — set it to `null` instead.
- Always update `last_indexed_commit` and `last_indexed_at` after `index.md` writes.
- Never create `components/<name>.md` unless the component has its own stack
  or its own relations.
- Append closed decisions only to an existing `decisions.md`; never create it
  outside onboarding.

## 5. Reference Rule

Specter, Linear, and CodeGraph are cited by ID or path only. If an agent
needs the actual content behind a reference, it resolves it against the
source at that moment. Never store a copy of that content in the vault.

## Result Contract

```
status: done | blocked | partial
executive_summary: one sentence
vault_changes: files created or edited, or none
field_changes: key: old -> new, per file
next_recommended: one concrete next step or none
risks: unresolved risks or none
```
