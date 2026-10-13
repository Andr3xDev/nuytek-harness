# abysal-harness

Personal Claude Code + OpenCode harness. Portable source to install the same agent
config on any of my machines.

## Install

```bash
./scripts/install.sh
```

After deploying this harness's configuration, the installer shallow-clones current
`main` from the upstream Caveman repository into a temporary directory and runs its
official installer for Claude Code and OpenCode. It requires `git` and `node`,
modifies user-level configuration, removes Cavecrew agents/skill, retains Caveman
plugin and communication skills, and removes the temporary clone on exit.

After copying harness configuration, interactive installs ask whether to register
`/home/andrex/dev/specter` as OpenSpec store `specter`. Registration runs only on
an explicit `yes`; non-interactive installs skip it. If selected, it requires
`openspec` and that store root. It fails if either is unavailable, or if `specter`
points to another root.

Installer preserves replaced targets in timestamped `<target>.backups/` directories. Remove
only installer backup directories without installing:

```bash
./scripts/install.sh --clean-backups
```

Update current repo into local config (smoke test, install, then remove installer backups):

```bash
./scripts/update.sh
```

Then set secrets/auth outside repo:

- `GITHUB_TOKEN` in shell env
- `ZENHUB_API_TOKEN` and `ZENHUB_WORKSPACE_ID` in shell env (optional, for ZenHub)
- Linear MCP auth
- Claude cloud connectors at `https://claude.ai/customize/connectors`
- `~/.local/bin` in `PATH` for `codegraph-health`

Validate:

```bash
claude doctor
opencode debug config
opencode mcp list
```

Refresh imports local config into repo; use `update.sh` to deploy repo into local config:

```bash
./scripts/refresh-from-local.sh
```

Refresh replaces managed paths exactly, excluding runtime data, caches, credentials, and
`.git`. Previous managed repo state is saved under gitignored `.refresh-backups/<timestamp>/`.

```bash
./scripts/refresh-from-local.sh --restore <timestamp>
./scripts/refresh-from-local.sh --clean-backups
./scripts/smoke-install.sh
```

Smoke always uses temporary `HOME`; installed CLI checks print `SKIP` when unavailable.

## System overview

```
tech-orchestrator          strategist
  (decomposes, delegates)    (business/product discussion, no code)
        │                         │
        ├── sdd-propose     → proposal.md; creates change directory ◄─ strategist may also delegate here
        ├── sdd-explore     → explore.md after proposal             ◄─ strategist may also delegate here
        ├── sdd-spec        → specs/{project}/{domain}/spec.md (ADDED/MODIFIED Requirements) ◄─ strategist may also delegate here
        ├── sdd-design      → design.md (ADR-lite)                 ◄─ strategist may also delegate here
        ├── sdd-tasks       → tasks.md (ordered, PR-size forecast)
        ├── builder         → default code/config/docs writer
        ├── quick-builder   → low-complexity edits (Haiku 5.5); blocked needs-builder if beyond scope
        ├── test-writer     → valuable failing tests (TDD red)
        ├── implementer     → strict TDD green from existing failing tests
        ├── code-reviewer   → spec/design/test conformance review (can run tests/linters/read-only git to verify claims)
        ├── judge-a/judge-b → blind dual adversarial review (judgment-day)
        ├── debugger        → root cause + proposed fix (investigate-only; also reads large logs)
        ├── codegraph-maintainer → CodeGraph index health
        ├── engram-maintainer → explicit memory review/curation
        └── sdd-verify → sdd-archive → change archived
```

The orchestrator never writes code directly. Strategist's delegation is scoped to
`sdd-explore`/`sdd-propose`/`sdd-spec`/`sdd-design` only — never to implementation-phase
agents (`sdd-tasks`, `builder`, `quick-builder`, `test-writer`, `implementer`, `code-reviewer`,
`sdd-verify`, `sdd-archive`). SDD activates only when the user explicitly asks for
plan/spec/design/SDD/OpenSpec — existing artifacts alone never activate it —
otherwise implementation goes straight to `quick-builder` (low-complexity) or `builder`, or the TDD loop (`test-writer` →
`implementer` → `code-reviewer`) with no spec required. TDD runs only when behavior value,
existing coverage, duplication/overlap, and protection of current behavior justify it; low-value,
duplicate/overlapping, stale, or disproportionate tests are rejected; otherwise
`builder` uses smallest useful proof. Bug regressions require meaningful externally observable behavior proportionate to risk.
Never delegate to external Cavecrew agents (`cavecrew-builder`, `cavecrew-investigator`,
`cavecrew-reviewer`); route only to named harness agents. Cavecrew is external tooling,
distinct from the Caveman communication skill/plugin.

Full SDD runs `sdd-propose` → `sdd-explore` → `sdd-spec` → `sdd-design` → `sdd-tasks`.
`sdd-propose` creates the change directory before `sdd-explore` persists `explore.md`.
When `.openspec.yaml` has `skip_specs: true`, omit `sdd-spec` and run `sdd-design`.
`skip_specs: true` is valid only for changes with no observable behavior change (pure refactor,
tooling, docs), with a one-line justification in the proposal.

Before any non-trivial delegation or change, orchestrators give a natural, contextual update with relevant detected issue, impact, planned action, affected area, validation, and real risk or blocker. They use no fixed labels or mandatory field list, omit irrelevant detail, avoid vague notices, and keep routine tiny reads/checks silent.

## Agents (`agents/`)

| Agent | Role |
|-------|------|
| `tech-orchestrator` | Claude entry point for multi-step task. Decomposes, delegates, never codes. |
| `strategist` | Business-language discussion partner — epics/goals, no code. May delegate, but only to `sdd-explore`/`sdd-propose`/`sdd-spec`/`sdd-design`; never to implementation-phase agents. |
| `sub-agents/sdd/*` | One agent per SDD phase: explore, propose, spec, design, tasks, verify, archive. |
| `sub-agents/general/builder` | Default writer when TDD value gate does not pass. |
| `sub-agents/general/quick-builder` | Low-complexity work (small edits, formatting, lint fixes, named dependency add/bump, config tweaks) on Haiku 5.5; returns blocked `needs-builder` when beyond scope. `explorer`, `codegraph-maintainer` and `sdd-archive` also run on Haiku 5.5. |
| `sub-agents/general/explorer` | Read-only exploration of many or large files for the orchestrator. |
| `sub-agents/tdd/test-writer` | Writes valuable failing tests from specs (red phase). |
| `sub-agents/tdd/implementer` | Minimal implementation to turn tests green. |
| `sub-agents/review/code-reviewer` | Spec/design/test conformance review. Has a scoped `Bash` tool (hook-restricted to running tests/linters and read-only git) to verify test/lint claims — still cannot modify any file. |
| `sub-agents/review/judge-a`, `judge-b` | Blind parallel adversarial review for critical features. |
| `sub-agents/debug/debugger` | Investigate-only root cause analysis, including large log reading; reports root cause, evidence, proposed fix, validation plan, and residual risk. Never edits; the fix goes to builder after user approval. |
| `sub-agents/infrastructure/aws` | Read-only AWS investigation. |
| `sub-agents/infrastructure/codegraph-maintainer` | Checks CodeGraph index status; init/sync/index only when explicit. |
| `sub-agents/infrastructure/engram-maintainer` | Reviews Engram memory; delete, purge, and export require native confirmation. |

Each SDD phase agent is an **executor**, not a sub-orchestrator: it does the
phase's work itself, never delegates further, and returns a structured
`status / executive_summary / artifacts / next_recommended / risks` envelope.

## Commands (`commands/`)

Slash-command entry points:

- `/explore`: read-only pre-check.
- `/review`: full standards review.
- `/pr-review`: drafts PR comments, never posts.
- `/clear-comments`, `/clear-tests`: report first, apply after OK.
- `/codegraph`, `/memory`: run their maintainer agents.
- `/echor`: frozen.

## CodeGraph Maintenance

Read-only health check:

```bash
./scripts/codegraph-health.sh /path/to/repo
codegraph-health /path/to/repo
```

Explicit maintenance:

```bash
./scripts/codegraph-health.sh --init /path/to/repo
./scripts/codegraph-health.sh --sync /path/to/repo
./scripts/codegraph-health.sh --index /path/to/repo
codegraph-health --sync /path/to/repo
```

## Out-Context Sessions

`claude-out-context [claude args]` starts Claude Code with the Engram plugin
disabled, Engram MCP tools denied, and a system prompt forbidding SDD and
tracker updates. Use it for work that must leave no memory trace (grading,
university tasks). The mode is fixed at launch; it cannot be toggled mid-session.

## Memory Maintenance

`/memory review [project]` and `/memory doctor [project]` are read-only. `/memory delete <observation-id>` inspects then soft-deletes one observation after native confirmation. `/memory purge <project>` hard-deletes a project after native confirmation. `/memory export [project]` writes an export only after native confirmation.

## Skills (`skills/`)

Reference material loaded on demand (not always-on context):

- `senior-architect/`, `software-design-patterns/`, `refactoring-techniques/` — architecture/design consultation
- `event-schema/` — validate domain events are schema'd before implementation
- `context-compact/` — preserve session state before `/compact` or `/clear`
- `judgment-day/` — blind dual-review protocol
- `find-docs/`, `karpathy-guidelines/` — library docs lookup, LLM-coding-mistake avoidance
- `md-style-guide/` — Markdown conventions and tracker body rules
- `github-issue/`, `github-pr/`, `linear-issue/`, `zenhub-issue/`, `release-notes/` — orchestrator-only tracker drafting (publish only after explicit user OK)
- `skill-registry/` — indexes skills by trigger phrase and path

## Configs (`configs/`)

Sanitized/generic versions of the global harness config (no user-specific
paths, tracker credentials, or personal notes — those live only in the
private `~/.claude/CLAUDE.md`):

- `generic-config.md` — engineering principles, agent commandments, delegation protocol
- `common-sdd.md` — shared executor-boundary protocol for all SDD phase agents
- `engram-protocol.md` — memory save/search triggers
- `mcp-servers.md` — MCP servers and plugins in use, no secrets
- `CLAUDE.md` — live Claude global instructions snapshot
- `claude-settings.json` — live Claude settings snapshot
- `claude-mcp.json` — Claude MCP servers merged into `~/.claude.json`
- `context7.md` — Context7 MCP rule

## MCP servers

| Server | Purpose |
|--------|---------|
| `engram` | Persistent memory across sessions (decisions, bugs, conventions) |
| `codegraph` | Symbol/call-graph index per project (`.codegraph/`) |
| `context7` | Up-to-date library/framework docs |
| `github` | Issues, PRs, repo search |
| `linear-server` | Linear issues/projects/initiatives |
| `zenhub` | ZenHub epics/goals/tasks (optional; needs `ZENHUB_API_TOKEN`, `ZENHUB_WORKSPACE_ID`) |
| `filesystem` | Sandboxed filesystem access outside project root |
Full config (types, commands, plugin sources) in `configs/mcp-servers.md`.

## Hooks (`hooks/`)

| Hook | Trigger | Purpose |
|------|---------|---------|
| `stop-verify.sh` | Stop | Verifies work before letting the agent report done |
| `herdr-agent-state.sh` | SessionStart | Reports session state to `herdr` (external agent monitor) |
| `*.sh` | Runtime hooks | Installed into `~/.claude/hooks/` by `scripts/install.sh` |

## Memory system

- Episodic/semantic: [engram](https://github.com/Gentleman-Programming/engram) — MCP-backed persistent memory across sessions
- Structural: [codegraph](https://github.com/colbymchenry/codegraph) — per-project symbol/call-graph index (`.codegraph/`)
- Procedural: `skills/` — on-demand reference material
- Spec/change history: [OpenSpec](https://github.com/Fission-AI/openspec) convention, artifacts stored in `~/dev/specter/openspec/`

### Spec path convention

```
openspec/
├── config.yaml
├── changes/
│   ├── {project}-{change-name}/
│   │   ├── proposal.md
│   │   ├── design.md
│   │   ├── tasks.md
│   │   └── specs/{project}/{domain}/spec.md
│   └── archive/YYYY-MM-DD-{project}-{change-name}/   # closed changes; openspec archive adds the date prefix
└── specs/{project}/{domain}/spec.md       # consolidated, updated on archive
```

`{project}` is the repo/service the change targets (e.g. `hyprland`, `nvim`,
`bridge-api`). Change IDs stay flat for OpenSpec.

## Communication/coding modes

Two independent, stackable plugin modes (default `full`, pinned in
`~/.config/{caveman,ponytail}/config.json`):

- **caveman** — governs *how the agent talks*: ultra-compressed prose, no filler
- **ponytail** — governs *how the agent codes*: YAGNI ladder, stdlib/native before custom, shortest working diff

See `configs/mcp-servers.md` for the full plugin/MCP list.

## Status Bars

- Claude Code: `claude-hud` (`jarrodwatts/claude-hud`) via `statusLine.command` in `configs/claude-settings.json`
- OpenCode TUI: `opencode-subagent-statusline` via `opencode/tui.json`
