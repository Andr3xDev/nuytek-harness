---
name: skill-registry
description: "Trigger: update skills, skill registry, refresh skills, or role skill policy. Index available skills and apply the role-scoped delegation policy."
license: MIT
metadata:
  author: custom
  version: "1.2"
---

## Purpose

This is the authoritative role-scoped skill-injection policy for both OpenCode and
Claude Code. It governs delegation prompts only; runtime plugin activation is separate.

## Hard Rules

- Pass exact paths to subagents; never paste skill summaries.
- Inject only the target role's mandatory paths and matching conditional paths.
- Do not inject Caveman, Ponytail, or Karpathy universally.
- Caveman role guidance is only for user-facing primaries: `orchestrator` and `strategist`.
- Never delegate to external Cavecrew agents (`cavecrew-builder`, `cavecrew-investigator`, `cavecrew-reviewer`); route only to named harness agents. Cavecrew is external tooling, distinct from Caveman.
- Ponytail is never injected into SDD, review, or infrastructure executors.
- Judgment-day belongs only to the orchestrator; never inject it into `judge-a` or `judge-b`.
- The SDD protocol is mandatory for the orchestrator and every `sdd-*` role.
- `md-style-guide` is mandatory for SDD artifact writers and conditional for other SDD roles when producing Markdown.

## Runtime Plugins

Runtime plugins are not delegation skills. Do not install, configure, enable, disable,
or otherwise alter them while applying this registry.

| Plugin | Role-policy effect |
|---|---|
| Caveman | Global runtime integration; do not inject a Caveman package path. Its role guidance applies only to `orchestrator` and `strategist`. |
| Ponytail | Inject its skill only where this registry permits it. |

## Mandatory Role Skills

`orchestrator` is OpenCode's primary name; `tech-orchestrator` is its Claude Code
mirror and follows the same row. The SDD protocol path is a shared protocol document,
not a runtime plugin.

| Role | Always inject |
|---|---|
| orchestrator / tech-orchestrator | caveman, karpathy-guidelines, SDD protocol |
| strategist | caveman, karpathy-guidelines |
| builder | ponytail |
| test-writer | ponytail |
| implementer | ponytail |
| debugger | karpathy-guidelines |
| code-reviewer | karpathy-guidelines |
| judge-a | karpathy-guidelines |
| judge-b | karpathy-guidelines |
| sdd-explore, sdd-verify | karpathy-guidelines, SDD protocol |
| sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-archive | karpathy-guidelines, SDD protocol, md-style-guide |
| aws, log-reader, codegraph-maintainer, engram-maintainer | none |
| echor-onboarder, echor-updater, echor-consultador, echor-validator | echor-vault |

## Conditional Role Skills

Apply a conditional skill only when both the role and task trigger match.

| Skill | Eligible roles | Task trigger |
|---|---|---|
| find-docs | orchestrator, strategist, builder, test-writer, implementer, debugger, sdd-explore, sdd-design, aws, echor-onboarder, echor-updater | Library, framework, SDK, API, CLI, cloud service, setup, migration, or API uncertainty. |
| refactoring-techniques | builder, implementer, debugger, code-reviewer | Refactor or behavior-preserving restructuring. |
| senior-architect | orchestrator, strategist, sdd-explore, sdd-propose, sdd-design | Architecture, system design, database choice, stack evaluation, dependency analysis, or trade-off comparison. |
| software-design-patterns | strategist, sdd-explore, sdd-design | Choosing, understanding, or identifying a design pattern. |
| event-schema | orchestrator, strategist, builder, test-writer, implementer, debugger, code-reviewer, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-verify | Domain event, event schema, event-driven behavior, or emitted event. |
| ponytail | debugger | `AUTH: autonomous-small-fix`; do not inject for `diagnose-only`. |
| context-compact | orchestrator, strategist | Context compaction, reset, clear, or state recovery. |
| skill-registry | orchestrator, builder | Skill installation, removal, creation, movement, rename, registry refresh, or role-policy change. |
| judgment-day | orchestrator | Adversarial review, dual review, judgment day, or critical review protocol. |
| md-style-guide | sdd-explore, sdd-verify | Producing Markdown. |

## Exact Paths

| Skill or protocol | Path |
|---|---|
| ponytail | `/home/andrex/.cache/opencode/packages/@dietrichgebert/ponytail@latest/node_modules/@dietrichgebert/ponytail/skills/ponytail/SKILL.md` |
| karpathy-guidelines | `/home/andrex/.config/opencode/skills/karpathy-guidelines/SKILL.md` |
| SDD protocol | `/home/andrex/dev/side-projects/harness-abysal/configs/common-sdd.md` |
| md-style-guide | `/home/andrex/.config/opencode/skills/md-style-guide/SKILL.md` |
| find-docs | `/home/andrex/.config/opencode/skills/find-docs/SKILL.md` |
| refactoring-techniques | `/home/andrex/.config/opencode/skills/refactoring-techniques/SKILL.md` |
| senior-architect | `/home/andrex/.config/opencode/skills/senior-architect/SKILL.md` |
| software-design-patterns | `/home/andrex/.config/opencode/skills/software-design-patterns/SKILL.md` |
| event-schema | `/home/andrex/.config/opencode/skills/event-schema/SKILL.md` |
| context-compact | `/home/andrex/.config/opencode/skills/context-compact/SKILL.md` |
| skill-registry | `/home/andrex/dev/side-projects/harness-abysal/opencode/skills/skill-registry/SKILL.md` |
| judgment-day | `/home/andrex/.config/opencode/skills/judgment-day/SKILL.md` |
| echor-vault | `/home/andrex/.config/opencode/skills/echor-vault/SKILL.md` |

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
