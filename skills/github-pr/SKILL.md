---
name: github-pr
description: "Trigger: PR description, pull request body, open a PR. Generate a standard GitHub pull request description from the diff."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use when the user asks to write, open, or rewrite a pull request description.
Read `../md-style-guide/SKILL.md` first; this skill adds only PR rules.

## Hard Rules

- If the repo has a PR template, its sections win.
- Use only when the branch is already pushed. Content comes from the pushed branch via GitHub MCP (`mcp__github__list_commits` with the head branch as `sha`, `mcp__github__get_pull_request_files` when the PR exists, `mcp__github__get_file_contents` if needed) — nothing outside it.
- Title follows Conventional Commits: `<type>(<scope>): <summary>`; type is one of feat|fix|refactor|perf|docs|test|build|ci|chore|revert; `!` before `:` marks a breaking change; imperative, lowercase summary, no trailing period.
- When the user asks, copy the title from the indicated commit or issue instead of writing a new one.
- Change type lives in the title and in GitHub labels, never in the body.
- `Changes` explains, not enumerates: what, why, and how it fits, grouped by area.
- `How to test` only claims results of commands that were actually run.
- Never run `git commit`, `git push`, `git add`, or `gh pr create`.
- Link issues with `Closes #123` or `Refs ENG-123`.
- A diagram, if any, goes in `Changes`, only when the change alters a flow.

## Decision Gates

| **Situation** | **Action** |
|:---|:---|
| Diff mixes unrelated changes | Flag it and suggest splitting before writing |
| Breaking change or migration | Keep `Reviewer notes` and state the migration |
| Base branch unknown | Ask; never guess |
| User asks to publish | Show title and body, wait for user OK, then `mcp__github__create_pull_request` |
| Default | `templates/pull-request.md` |
