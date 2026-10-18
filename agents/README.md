<h1 align="center">Claude Code Sub-Agents</h1>

<p align="center">Multi-agent engineering harness — agent roster, delegation rules, and enforcement</p>

<br>

## What is this directory

This directory holds the sub-agent definitions used by Claude Code for the multi-agent engineering harness: `strategist.md`, `tech-orchestrator.md`, and the phase agents under `sub-agents/sdd/` (`sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-verify`, `sdd-archive`), plus `builder`, `quick-builder`, `git-ops`, `implementer`, `test-writer`, `explorer`, `code-reviewer`, `judge-a`/`judge-b`, infrastructure agents (`aws`, `codegraph-maintainer`, `engram-maintainer`), and the vault agents under `sub-agents/echor/` (`echor-onboarder`, `echor-updater`, `echor-consultador`, `echor-validator`).

OpenCode is on standby and is not maintained now; Claude Code is the active source of truth for this directory.

TDD runs only when behavior value, existing coverage, duplication/overlap, and protection of current behavior justify it. Low-value, duplicate/overlapping, stale, or disproportionate tests are rejected. `test-writer` runs a mandatory Phase 0 test inventory before RED. Otherwise `builder` handles work with the smallest useful proof. Bug regressions require meaningful externally observable behavior proportionate to risk.

SDD is proposed by the orchestrator and runs only after user approval; small fixes with no background skip it. Judgment-day runs only when the user explicitly asks.

<br>

## How SDD changes get archived

`sdd-archive` (`/home/andrex/.claude/agents/sub-agents/sdd/sdd-archive.md`) closes a change using the real `openspec` CLI instead of moving folders by hand:

```bash
openspec archive "<project>-<change-name>" -y --store specter
```

Before archiving, the agent requires `{change-folder}/verify-report.md` and refuses to proceed if it is missing or has CRITICAL findings.

If the CLI archive command is unavailable or fails, archive blocks. `sdd-archive` reports the command and failure to the orchestrator; it never creates directories or moves change folders manually.

Change-name convention (`{project}-{change-name}`, kebab-case, hyphens only) is defined once at `/home/andrex/dev/specter/openspec/AGENTS.md` and validated by `sdd-propose` before a change is created — this README does not duplicate that rule.

SDD commands use registered store `specter` from any working directory: `openspec <subcommand> ... --store specter`. Check registration with `openspec store list`; global structural validation is `openspec validate --all --strict --store specter`.

<br>

## How the strategist delegates

`strategist.md` has a "Delegation scope (inviolable)" section (around lines 98-110) restricting it to four SDD validation agents:

- `sdd-explore`
- `sdd-propose`
- `sdd-spec`
- `sdd-design`

The strategist may never invoke `Task` toward `implementer`, `test-writer`, `builder`, `quick-builder`, `sdd-tasks`, `sdd-verify`, or `sdd-archive` — those are execution-phase agents owned by the orchestrator. If a discussion reaches that phase, the strategist hands off to the orchestrator instead of delegating itself.

This boundary is a prompt-level lock, not a technical one — Claude Code's `Task` tool has no built-in mechanism to restrict which target agent a given caller can invoke. Enforcement depends entirely on the agent's own instructions being followed.

<br>

## How skills get injected

The authoritative role policy lives in `skills/skill-registry/SKILL.md`. Its mandatory assignments are:

| Agent | Mandatory skill |
|---|---|
| builder | ponytail |
| quick-builder | none |
| git-ops | caveman-commit |
| implementer | ponytail |
| test-writer | ponytail |
| orchestrator, strategist | caveman, karpathy-guidelines |
| sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-verify, sdd-archive | karpathy-guidelines, SDD protocol, md-style-guide |
| debugger, explorer, code-reviewer, judge-a, judge-b | karpathy-guidelines |
| aws, codegraph-maintainer, engram-maintainer | none — infrastructure |
| echor-onboarder, echor-updater, echor-consultador, echor-validator | echor-vault |

`caveman` is only injected into user-facing primaries (`orchestrator`, `strategist`).
`ponytail` is mandatory only for code writers (`builder`, `implementer`, `test-writer`). `quick-builder` is a code writer that receives no skills.
It is never injected into SDD, review, debugger, explorer, or infrastructure executors. `karpathy-guidelines`
is mandatory for orchestration, strategy, SDD, review, exploration, and debugging.

Never delegate to external plugin agents (`cavecrew-builder`, `cavecrew-investigator`,
`cavecrew-reviewer`, or any agent from the Caveman or Ponytail plugins); route only to named
harness agents. Caveman and Ponytail are used only as skills, never as agents.

Conditional skills (`find-docs`, `refactoring-techniques`, `senior-architect`,
`software-design-patterns`, `event-schema`, `context-compact`, `skill-registry`, and the
tracker skills `github-issue`, `github-pr`, `linear-issue`, `zenhub-issue`, `release-notes`)
are injected only for roles and tasks listed in the registry. Judgment-day belongs only
to the orchestrator, only on explicit user request, never the judges. Runtime plugin activation is separate from this
delegation policy.

<br>

## Explorer

`explorer` (`sub-agents/general/explorer.md`) is a read-only agent (haiku-5-5, effort xhigh; tools Read/Glob/Grep, codegraph, engram-read; no Bash). The orchestrator delegates reading to it when a task needs 4+ files or one very large file. It returns only what is needed, with `path:line` evidence. It is distinct from `sdd-explore`, which belongs to the SDD flow and writes `explore.md`.

<br>

## Echor vault agents

`sub-agents/echor/` mirrors the four agents that maintain Echor, the Obsidian vault at `~/dev/echor` centralizing knowledge about the user's 100+ repos. All four load the shared `echor-vault` skill (vault paths, naming, frontmatter schema, read recipes, write rules) before doing any work.

| Agent | Role | Writes |
|---|---|---|
| echor-onboarder | Creates the vault entry for a repo that has none | `~/dev/echor/projects/<slug>/index.md`, `decisions.md` |
| echor-updater | Detects what changed and updates an existing index or appends a closed local decision | edits existing `index.md`, `decisions.md` |
| echor-consultador | Read-only query service over the vault | nothing, ever |
| echor-validator | Read-only consistency audit, vault versus real code | nothing, report only |

`echor-onboarder` and `echor-validator` share the same read-only Git and inspection allow-list (`git status*`, `git log*`, `git diff*`, `git show*`, `git rev-parse*`, `ls *`, `rg *`, `cat *`); commit-creating and staging git and `git push` are denied, and `git reset --hard*`, `git clean -f*`, `git checkout -- *`, `git restore *`, `git stash*` require confirmation. `echor-updater` uses the same Bash allow-list but only `edit` (no `write`), since it only ever modifies an existing note. `echor-consultador` has no Bash access at all — `read`/`glob`/`grep` only, nothing else.

`/echor` (`claude/commands/echor.md`) routes to the four agents by subcommand: `onboard <repo-path>`, `update <slug|all>`, `close <slug> <decision summary>`, `ask <question>`, `validate [slug|all]`.

<br>

## Enforcement of permissions (AUTH gates)

Claude Code has no per-agent command allow-list built into agent frontmatter beyond `tools`/`disallowedTools`. The gap is closed with `PreToolUse` hooks registered in `settings.json`.

`/home/andrex/.claude/hooks/infra-agent-bash-guard.py` is the reference implementation of this pattern. It intercepts `Bash` calls, keyed on `agent_type`. SDD agents are limited to documented `openspec` and read-only Git commands; prompt scope limits each artifact writer to its own file. Reviewers and judges can run tests, linters, and read-only Git without gaining open shell access.

Commit-creating and staging git is denied for every agent except git-ops; push is denied for all. The hook never returns `ask` (a hook `ask` in a background subagent never reaches the user): discards and `git stash` prompt through native `permissions.ask`, and odd invocation forms (`git -C`, absolute path, `env git`, chained) are denied, so use the plain `git <sub>` form. Read-only agents may not discard at all. The Ask column lists commands the hook passes through to native `permissions.ask`.

| Agent | Allow | Ask | Deny |
|---|---|---|---|
| aws | `aws logs describe-*`, `aws dynamodb *`, `aws lambda get-*`/`list-*`, `aws ecs describe-*`/`list-*`, `aws ec2 describe-*`, `git *` | none | none |
| codegraph-maintainer | `codegraph status/query/explore/files/node/callers/callees/impact/affected`, `git *` | `codegraph unlock/init/sync/index*`, `codegraph-health*` | none |
| debugger | Same test/lint/build commands as code-reviewer; read-only Git; `rg`, `wc`, `du`, `ls`, `cat`, `head`, `tail`, `zcat`, `gzip -cd`, `journalctl`, `docker logs`, `kubectl logs` | none | package install/update/sync/add/remove/publish, arbitrary scripts, shell chaining, other shell commands |
| engram-maintainer | `engram context/search/stats/projects list/doctor/timeline` | `engram delete *`, `engram export *` | all unrelated shell commands |
| code-reviewer | Scoped Node, Python, Java/Kotlin, Go, Rust, .NET, Ruby, PHP, and `make` test/lint/format-check/typecheck/compile/build/static-analysis commands; read-only Git | none | package install/update/sync/add/remove/publish, arbitrary scripts, shell chaining, other shell commands |
| judge-a, judge-b | Same test/lint and read-only Git commands as `code-reviewer` | none | none |
| sdd-explore | `openspec context`/`doctor`/`list` with `--store specter`, `openspec store list`, read-only Git; native `Write`/`Edit` for prompt-scoped `explore.md` | none | all other shell commands |
| sdd-propose, sdd-spec, sdd-design, sdd-tasks | `openspec status`/`validate`/`show`/`new`/`instructions` with `--store specter`, `openspec store list`, read-only Git | none | all other shell commands |
| sdd-verify | `openspec status`/`validate` with `--store specter`, `openspec store list`, reviewer test/lint, read-only Git; native `Write`/`Edit` for prompt-scoped `verify-report.md` | none | all other shell commands |
| sdd-archive | `openspec status`/`validate`/`show`/`instructions`/`archive` with `--store specter`, `openspec store list`, read-only Git | none | all other shell commands |
| echor-onboarder, echor-validator | Read-only Git (`status`/`log`/`diff`/`show`/`rev-parse`), `ls *`, `rg *`, `cat *` | none | all other shell commands |
| echor-updater | Same read-only Git and inspection commands as `echor-onboarder` | none | all other shell commands |
| echor-consultador | none | none | all shell commands |

Any command not matched by `allow` or `ask` falls through to `deny`. Any `agent_type` not in the `RULES` dict (the main thread and git-ops included) skips the allow-list, but still gets the git rules: staging denied except for git-ops, push denied, discards and stash passed to native ask in plain form (denied in odd forms), and git-ops may not run `rebase -i`.

This is the pattern to reuse if another agent needs a Bash restriction in the future — for example, `test-writer` arguably should not touch production code paths. Today that boundary is prompt discipline only (documented in `test-writer`'s own instructions); no hook enforces it. This is a known gap, not yet addressed.

### `debugger` investigate-only policy

`debugger` reads logs, investigates, and reports root cause, evidence, proposed fix, validation plan, and residual risk. It never edits; the fix goes to `builder` after user approval. It also covers large log reading (formerly `log-reader`). Existing destructive-command confirmation and the git staging/push denial still apply.

<br>

## Circuit breaker

`tech-orchestrator.md`, "Validation and close" section: if `code-reviewer` reports the same BLOCKER on the same finding/file 3 times in a row, the orchestrator stops the reviewer-implementer cycle and escalates to the human instead of iterating further.

<br>

## Parity between Claude Code and OpenCode

OpenCode is on standby. Claude Code is the active source of truth; parity between the two is not maintained until OpenCode is resumed.
