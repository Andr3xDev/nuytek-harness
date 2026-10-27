<h1 align="center">Claude Code Sub-Agents</h1>

<p align="center">Multi-agent engineering harness — agent roster, delegation rules, and enforcement</p>

<br>

## What is this directory

This directory holds the sub-agent definitions used by Claude Code for the multi-agent engineering harness: `strategist.md`, `tech-orchestrator.md`, and the phase agents under `sub-agents/sdd/` (`sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-verify`, `sdd-archive`), plus `builder`, `quick-builder`, `git-ops`, `implementer`, `test-writer`, `explorer`, `code-reviewer`, `judge-a`/`judge-b`, infrastructure agents (`aws`, `codegraph-maintainer`, `engram-maintainer`).

TDD runs only when behavior value, existing coverage, duplication/overlap, and protection of current behavior justify it. Low-value, duplicate/overlapping, stale, or disproportionate tests are rejected. `test-writer` runs a mandatory Phase 0 test inventory before RED. Otherwise `builder` handles work with the smallest useful proof. Bug regressions require meaningful externally observable behavior proportionate to risk.

SDD is proposed by the orchestrator and runs only after user approval; small fixes with no background skip it. Judgment-day runs only when the user explicitly asks.

<br>

## How SDD changes get archived

`sdd-archive` (`/home/andrex/.claude/agents/sub-agents/sdd/sdd-archive.md`) closes a change using the real `openspec` CLI instead of moving folders by hand:

```bash
openspec archive "<change>" --yes --store <store-id>
```

Before archiving, the agent requires `{change-folder}/verify-report.md` and every task in `tasks.md` complete, and refuses to proceed if the report is missing, has CRITICAL findings, or tasks are open (`--yes` would bypass the CLI's own check).

If the CLI archive command is unavailable or fails, archive blocks. `sdd-archive` reports the command and failure to the orchestrator; it never creates directories or moves change folders manually.

Change-name convention (kebab-case, hyphens only; `{project}-{change-name}` only in the shared default store, no project prefix in a product store) is defined once at `/home/andrex/dev/specter/openspec/AGENTS.md` and validated by `sdd-propose` before a change is created — this README does not duplicate that rule.

SDD commands use a registered store from any working directory: `openspec <subcommand> ... --store <store-id>`. The orchestrator resolves the id from the repo's `openspec/config.yaml` `store:` (default `specter`) and passes `STORE: <store-id>` in every delegation. Check registration with `openspec store list`; global structural validation is `openspec validate --all --strict --store <store-id>`. Creating or registering stores is never an SDD agent task: the orchestrator asks the user and delegates to `builder`. After an SDD block that wrote artifacts, `git-ops` commits inside the store root.

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

`explorer` (`sub-agents/general/explorer.md`) is a read-only agent (haiku-5-5, effort xhigh; tools Read/Glob/Grep/Bash, codegraph, engram-read; Bash limited by the guard to read-only fs and git read commands). The orchestrator delegates reading to it when a task needs 4+ files or one very large file. It returns only what is needed, with `path:line` evidence. It is distinct from `sdd-explore`, which belongs to the SDD flow and writes `explore.md`.

<br>

## Enforcement of permissions (AUTH gates)

Claude Code has no per-agent command allow-list built into agent frontmatter beyond `tools`/`disallowedTools`. The gap is closed with `PreToolUse` hooks registered in `settings.json`.

`/home/andrex/.claude/hooks/infra-agent-bash-guard.py` is the reference implementation of this pattern. It intercepts `Bash` calls, keyed on `agent_type`. SDD agents are limited to documented `openspec` and read-only Git commands; prompt scope limits each artifact writer to its own file. Reviewers and judges can run tests, linters, and read-only Git without gaining open shell access.

Each agent's allow-list is composed from named roles (plain lists in the guard): `ROLE_READ_FS` (`ls`, `tree`, `wc`, `du`, `cat`, `head`, `tail`, `grep`, `rg`), `ROLE_GIT_READ`, `ROLE_GH_READ`, `ROLE_VALIDATE`, `ROLE_LOGS`, and the `ROLE_OPENSPEC_*` groups. `explorer` = fs + git read + openspec read; `sdd-explore` = openspec read + git read + fs read; `debugger` = validate + git/gh read + fs read + logs. `rg --pre`/`--pre-glob` (arbitrary program execution) is denied for every guarded agent, as are `tree` write options (`--output`, `-o`, `-R`) and git `--output`, `--ext-diff`, `--textconv`, plus secret files (`.env*` except `.env.example`/`.sample`/`.template`, `*.pem`, `*.key`, `.ssh/`, `.aws/`, ...), recursive `grep`, and `rg` with `--hidden`/`--no-ignore*`/`-u`/`-.`.

Commit-creating and staging git is denied for every agent except git-ops; push is denied for all but git-ops, and only when the user asked; force push prompts through native `permissions.ask`. Worktree mutations (`add`, `remove`, `prune`, `move`, `lock`, `unlock`, `repair`) are denied for all but git-ops, and worktree/branch deletes prompt through native `permissions.ask`; `git worktree list` stays read-only. The hook returns `ask` for file deletion (`rm`, `rmdir`, `unlink`, `shred`, `find -delete`, `find -exec rm`), for every agent, and for risky git in `git -C <absolute path>` form (force/delete push, discards, `stash`, worktree/branch deletes) that native rules cannot match; deny still wins. Any agent may use `git -C <absolute path>` for the git its role allows; a relative `-C` path is denied. Plain-form discards and `git stash` prompt through native `permissions.ask`, and other odd forms (`env git`, chained) are denied. Read-only agents may not discard at all. The Ask column lists commands the hook passes through to native `permissions.ask`.

| Agent | Allow | Ask | Deny |
|---|---|---|---|
| explorer | `ls`, `tree`, `wc`, `du`, `cat`, `head`, `tail`, `grep`, `rg` (ROLE_READ_FS); read-only git (ROLE_GIT_READ) | none | `rg --pre`/`--pre-glob`, `tree` write options, secret-file reads, recursive `grep`, `rg` with `--hidden`/`--no-ignore*`/`-u`/`-.`, other shell commands |
| aws | `aws logs describe-*`, `aws dynamodb` `describe-*`/`list-*`/`get-item`/`query`/`scan`, `aws lambda get-*`/`list-*`, `aws ecs describe-*`/`list-*`, `aws ec2 describe-*`, read-only git (ROLE_GIT_READ) | none | none |
| codegraph-maintainer | `codegraph status/query/explore/files/node/callers/callees/impact/affected`, read-only git (ROLE_GIT_READ) | `codegraph unlock/init/sync/index/uninit*`, `codegraph-health*` | none |
| debugger | Same test/lint/build commands as code-reviewer; read-only Git; read-only `gh` (`run view`/`list`, `pr view`/`diff`/`checks`/`list`, `issue view`, `workflow view`); `rg`, `wc`, `du`, `ls`, `cat`, `head`, `tail`, `zcat`, `gzip -cd`, `journalctl`, `docker logs`, `kubectl logs` | none | package install/update/sync/add/remove/publish, running arbitrary scripts directly (project code runs only through test/lint/build targets), shell chaining, other shell commands |
| engram-maintainer | `engram context/search/stats/projects list/doctor/timeline/export`, soft `engram delete` | `engram delete *--hard*` (only on an explicit user request for that hard delete) | all unrelated shell commands |
| code-reviewer | Scoped Node (incl. Vitest/Jest/`node --test`), QML (`qmllint` and the `qml6`/`qmlscene`/`qmltestrunner` runners), Python, Java/Kotlin, Go, Rust, .NET, Ruby, PHP, Terraform/Terragrunt (`fmt -check`/`validate`/`tflint`/`hcl validate`), and `make` test/lint/format-check/typecheck/compile/build/static-analysis commands; read-only Git; read-only `gh` (`pr view`/`diff`/`checks`/`list`, `run view`, `issue view`) | none | package install/update/sync/add/remove/publish, running arbitrary scripts directly (project code runs only through test/lint/build targets), shell chaining, other shell commands |
| judge-a, judge-b | Same test/lint and read-only Git commands as `code-reviewer` | none | none |
| sdd-explore | ROLE_OPENSPEC_READ (`openspec list`/`show`/`status`/`validate`/`instructions`/`context`/`doctor`/`store list`) with `--store <store-id>`, read-only Git, read-only fs commands (ROLE_READ_FS); native `Write`/`Edit` for prompt-scoped `explore.md` | none | all other shell commands, including `openspec store setup|register|unregister|remove`, `init`, `update`, `workset` |
| sdd-propose, sdd-spec, sdd-design, sdd-tasks | ROLE_OPENSPEC_READ plus `openspec new` with `--store <store-id>`, read-only Git | none | all other shell commands |
| sdd-verify | ROLE_OPENSPEC_READ with `--store <store-id>`, reviewer test/lint, read-only Git; native `Write`/`Edit` for prompt-scoped `verify-report.md` | none | all other shell commands |
| sdd-archive | ROLE_OPENSPEC_READ plus `openspec archive` with `--store <store-id>`, read-only Git, read-only fs commands (ROLE_READ_FS); native `Write`/`Edit` for `archive-report.md` | none | all other shell commands |

Runner globs execute project code (same trust as `npm test`); terragrunt `hcl validate` also evaluates `run_cmd()` in configs.
`git -C <abs>` reads in another repo run that repo's config (`core.fsmonitor`, `core.pager`), so only point agents at trusted repos.

Any command not matched by `allow` or `ask` falls through to `deny`. Any `agent_type` not in the `RULES` dict (the main thread and git-ops included) skips the allow-list, but still gets the git rules: staging and worktree mutations denied except for git-ops, push denied except for git-ops in plain or absolute `-C` form (force push goes to native ask or the hook's ask), discards and stash passed to native ask in plain form and asked by the hook in absolute `-C` form (denied in odd forms), and git-ops may not run `rebase -i`.

This is the pattern to reuse if another agent needs a Bash restriction in the future — for example, `test-writer` arguably should not touch production code paths. Today that boundary is prompt discipline only (documented in `test-writer`'s own instructions); no hook enforces it. This is a known gap, not yet addressed.

### `debugger` investigate-only policy

`debugger` reads logs, investigates, and reports root cause, evidence, proposed fix, validation plan, and residual risk. It never edits; the fix goes to `builder` after user approval. It also covers large log reading (formerly `log-reader`). Existing destructive-command confirmation and the git staging and push denial still apply.

<br>

## Circuit breaker

`tech-orchestrator.md`, "Validation and close" section: if `code-reviewer` reports the same BLOCKER on the same finding/file 3 times in a row, the orchestrator stops the reviewer-implementer cycle and escalates to the human instead of iterating further.
