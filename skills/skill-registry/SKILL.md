---
name: skill-registry
description: "Trigger: update skills, skill registry, refresh skills, or role skill policy. Index available skills and apply the role-scoped delegation policy."
license: MIT
metadata:
  author: custom
  version: "1.4"
---

## Purpose

This is the authoritative role-scoped skill-injection policy for
Claude Code. It governs delegation prompts only; runtime plugin activation is separate.

## Hard Rules

- Pass exact paths to subagents; never paste skill summaries.
- Inject only the target role's mandatory paths and matching conditional paths.
- Do not inject Caveman, Ponytail, or Karpathy universally.
- Caveman role guidance is only for user-facing primaries: `tech-orchestrator` and `strategist`.
- Never delegate to external plugin agents (`cavecrew-builder`, `cavecrew-investigator`, `cavecrew-reviewer`, or any agent from the Caveman or Ponytail plugins); route only to named harness agents. Caveman and Ponytail are skills only, never agents.
- Ponytail is never injected into SDD, review, or infrastructure executors.
- Judgment-day belongs only to the tech-orchestrator, only when the user explicitly asks for it; never inject it into `judge-a` or `judge-b`.
- The SDD protocol is mandatory for every `sdd-*` role; inject it for the tech-orchestrator only when the user approved SDD.
- Tracker skills (`linear-issue`, `github-issue`, `github-pr`, `release-notes`, `zenhub-issue`) are never injected into subagents: they ask the user and need tracker tools.
- `md-style-guide` is mandatory for SDD artifact writers and conditional for other SDD roles when producing Markdown.

## Runtime Plugins

Runtime plugins are not delegation skills. Do not install, configure, enable, disable,
or otherwise alter them while applying this registry.

| Plugin | Role-policy effect |
|---|---|
| Caveman | Global runtime integration; do not inject a Caveman package path. Its role guidance applies only to `tech-orchestrator` and `strategist`. |
| Ponytail | Inject its skill only where this registry permits it. |

## Mandatory Role Skills

The SDD protocol path is a shared protocol document,
not a runtime plugin.

| Role | Always inject |
|---|---|
| tech-orchestrator | caveman, karpathy-guidelines (+ SDD protocol only when the user approved SDD) |
| strategist | caveman, karpathy-guidelines |
| builder | ponytail |
| quick-builder | none |
| git-ops | caveman-commit |
| test-writer | ponytail |
| implementer | ponytail |
| debugger | karpathy-guidelines |
| explorer | karpathy-guidelines |
| code-reviewer | karpathy-guidelines |
| judge-a | karpathy-guidelines |
| judge-b | karpathy-guidelines |
| sdd-explore, sdd-verify | karpathy-guidelines, SDD protocol |
| sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-archive | karpathy-guidelines, SDD protocol, md-style-guide |
| aws, codegraph-maintainer, engram-maintainer | none |
| echor-onboarder, echor-updater, echor-consultador, echor-validator | echor-vault |

## Conditional Role Skills

Apply a conditional skill only when both the role and task trigger match.

| Skill | Eligible roles | Task trigger |
|---|---|---|
| find-docs | tech-orchestrator, strategist, builder, test-writer, implementer, debugger, sdd-explore, sdd-design, aws, echor-onboarder, echor-updater | Library, framework, SDK, API, CLI, cloud service, setup, migration, or API uncertainty. |
| refactoring-techniques | builder, implementer, debugger, code-reviewer | Refactor or behavior-preserving restructuring. |
| senior-architect | tech-orchestrator, strategist, sdd-explore, sdd-propose, sdd-design | Architecture, system design, database choice, stack evaluation, dependency analysis, or trade-off comparison. |
| software-design-patterns | strategist, sdd-explore, sdd-design | Choosing, understanding, or identifying a design pattern. |
| event-schema | tech-orchestrator, strategist, builder, test-writer, implementer, debugger, code-reviewer, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-verify | Domain event, event schema, event-driven behavior, or emitted event. |
| context-compact | tech-orchestrator, strategist | Context compaction, reset, clear, or state recovery. |
| skill-registry | tech-orchestrator, builder | Skill installation, removal, creation, movement, rename, registry refresh, or role-policy change. |
| judgment-day | tech-orchestrator | User explicitly asks for adversarial review, dual review, judgment day, or critical review protocol. |
| md-style-guide | tech-orchestrator, sdd-explore, sdd-verify | Producing Markdown or tracker bodies. |
| linear-issue | tech-orchestrator | Draft or create a Linear issue or project. |
| github-issue | tech-orchestrator | GitHub issue in a repo without a Linear team. |
| github-pr | tech-orchestrator | Write, open, or rewrite a PR title or description. |
| release-notes | tech-orchestrator | Release notes, GitHub release body, changelog entry, or chat release message. |
| zenhub-issue | tech-orchestrator | ZenHub epic, goal, or task. |

## Exact Paths

| Skill or protocol | Path |
|---|---|
| ponytail | Resolve with Glob `/home/andrex/.claude/plugins/cache/ponytail/ponytail/*/skills/ponytail/SKILL.md` and pass the highest version's exact path (the version changes on plugin updates) |
| caveman-commit | Resolve with Glob `/home/andrex/.claude/plugins/cache/caveman/caveman/*/skills/caveman-commit/SKILL.md` and pass the highest version's exact path (the version changes on plugin updates) |
| karpathy-guidelines | `/home/andrex/.claude/skills/karpathy-guidelines/SKILL.md` |
| SDD protocol | `/home/andrex/.claude/common-sdd.md` |
| md-style-guide | `/home/andrex/.claude/skills/md-style-guide/SKILL.md` |
| find-docs | `/home/andrex/.claude/skills/find-docs/SKILL.md` |
| refactoring-techniques | `/home/andrex/.claude/skills/refactoring-techniques/SKILL.md` |
| senior-architect | `/home/andrex/.claude/skills/senior-architect/SKILL.md` |
| software-design-patterns | `/home/andrex/.claude/skills/software-design-patterns/SKILL.md` |
| event-schema | `/home/andrex/.claude/skills/event-schema/SKILL.md` |
| context-compact | `/home/andrex/.claude/skills/context-compact/SKILL.md` |
| skill-registry | `/home/andrex/.claude/skills/skill-registry/SKILL.md` |
| judgment-day | `/home/andrex/.claude/skills/judgment-day/SKILL.md` |
| echor-vault | `/home/andrex/.claude/skills/echor-vault/SKILL.md` |
| linear-issue | `/home/andrex/.claude/skills/linear-issue/SKILL.md` |
| github-issue | `/home/andrex/.claude/skills/github-issue/SKILL.md` |
| github-pr | `/home/andrex/.claude/skills/github-pr/SKILL.md` |
| release-notes | `/home/andrex/.claude/skills/release-notes/SKILL.md` |
| zenhub-issue | `/home/andrex/.claude/skills/zenhub-issue/SKILL.md` |

## Delegation Procedure

1. Identify the target role.
2. Add every available exact path from that role's mandatory row. Caveman is supplied by the global runtime, not an injected package path.
3. Add only conditional paths whose role and trigger both match.
4. Put the resulting exact paths under `## Skills to load before work` in the delegation prompt.
5. If no paths apply, include an empty `SKILLS:` field; do not invent a default skill.

## Registry Maintenance

When skills are installed, removed, created, moved, or renamed, scan global and
project `*/SKILL.md` directories, prefer project skills over duplicate globals, skip
`_shared`, and update the generated registry index. Save the resulting index to Engram
with `topic_key: skill-registry`.
