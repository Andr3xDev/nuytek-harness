---
name: senior-architect
description: Reference skill for software architecture decisions. Trigger when the user asks about architecture patterns, system design, database selection, tech stack evaluation, dependency analysis, or trade-off comparisons. This is a consultation tool — read the relevant reference, present findings, and wait for the user's direction before acting.
---
# Senior Architect

Architecture reference for making informed technical decisions together with the user.

---

## Scope

**Covers:** architecture patterns, system design workflows, technology selection, dependency analysis, architecture diagrams.

**Does NOT cover:** code-level conventions (see `sainapsis-conventions`), CI/CD pipeline config, infrastructure provisioning.

---

## Preconditions

- The user has stated an architecture question, design decision, or wants to analyze an existing system.
- If the question is about code style or testing patterns, this is the wrong skill.

---

## Navigation Rule

1. Identify which topic matches the user's question using the reference table below.
2. Read **only** the relevant file — do not load everything.
3. Present what you found: the relevant pattern, trade-offs, or decision matrix.
4. **Wait for the user's confirmation or direction before applying anything.**

---

## References

| File | What it covers | Read when the user asks about |
|------|---------------|-------------------------------|
| `references/architecture_patterns.md` | 9 patterns with trade-offs and when to use each | "which pattern?", "monolith vs microservices", "event-driven", "CQRS", "hexagonal" |
| `references/system_design_workflows.md` | Step-by-step workflows for design tasks | "how to design X?", "capacity planning", "API design", "migration strategy" |
| `references/tech_decision_guide.md` | Decision matrices for technology choices | "which database?", "which cache?", "which message queue?", "which framework?" |

---

## Available Scripts

These scripts analyze a project's structure. **Do not run them without the user's approval.**

| Script | Purpose |
|--------|---------|
| `scripts/architecture_diagram_generator.py` | Generate diagrams (Mermaid, PlantUML, ASCII) from project structure |
| `scripts/dependency_analyzer.py` | Detect circular dependencies, coupling, outdated packages |
| `scripts/project_architect.py` | Assess architecture patterns and code organization issues |

Usage: `python scripts/<script>.py <project-path> [--format mermaid] [--verbose] [--output json]`

---

## Control Points

- **Before running any script:** describe what it will analyze and ask for approval.
- **After reading a reference:** summarize the relevant finding and wait — do not jump to implementation.
- **Before recommending a pattern or technology:** present the trade-offs and let the user decide.
- **If multiple patterns could apply:** list the options with pros/cons, do not pick one.

---

## Constraints

- Do not generate architecture recommendations from general knowledge alone — ground them in the reference files.
- Do not run scripts autonomously. Always ask first.
- Do not dump entire reference files as output. Extract only the relevant section.
- Keep responses concise. The user reads the trade-offs and decides — you assist, you don't dictate.
