# SDD Phase — Common Protocol

Boilerplate shared across all SDD phase agents.
Sub-agents MUST follow this alongside their phase-specific instructions.

---

## A. Executor boundary

Every SDD phase agent is an EXECUTOR, not an orchestrator.
Do the phase work yourself. Do NOT launch sub-agents, do NOT call the Task tool,
and do NOT bounce work back unless the phase explicitly says to stop and report a blocker.

---

## B. Skill loading

1. Check if the orchestrator injected a `## Skills to load before work` block in your
   delegation prompt. If yes, read those exact SKILL.md files before starting.
2. If no skills block was provided, proceed with your phase instructions only.
3. The orchestrator resolves skills — sub-agents do not self-discover.

---

## C. Artifact retrieval (OpenSpec filesystem)

Full SDD runs only for work the user approved as SDD. Its artifacts live only in the OpenSpec change directory
of the store named in your delegation: `{store-root}/openspec/changes/{change-name}/`.

The orchestrator passes `STORE: <store-id>` and the store root path in every SDD delegation CONTEXT. It resolves the id from a local `openspec/config.yaml` pointer or specter's `repos.yaml` (see "SDD store resolution" in `tech-orchestrator.md`).
Always pass `--store <store-id>` explicitly to every `openspec` command. Never `cd` into the
store and never rely on the working directory. Agents never run `openspec store setup|register|unregister`
or `openspec init|update|workset`; if a store is missing, return `status: blocked`.

Change naming: the store is the product, so the change name has no project prefix. Changes are flat
under `changes/`; nested change folders are not supported. A change may start with a temporary name
and be renamed later by moving its folder (no CLI rename). Never rename a store id: repo `store:`
pointers and `references:` depend on it.

Spec depth and duplicate check: section D.

Full route: `sdd-propose` → `sdd-explore` → `sdd-spec` → `sdd-design` → `sdd-tasks`.
`sdd-propose` creates the change directory before `sdd-explore` persists `explore.md`.
When `skip_specs: true`, omit `sdd-spec` and run `sdd-design`.

`skip_specs: true` is valid only for changes with no observable behavior change (pure
refactor, tooling, docs) and the proposal must include a one-line justification. Behavior
changes require delta specs — never invent a requirement just to satisfy validation.

Read every complete artifact file needed for the phase. When `.openspec.yaml` has
`skip_specs: true`, `specs/` is intentionally absent; use `proposal.md`, `design.md`,
and `tasks.md` as source of truth. Do not use Engram to retrieve SDD artifacts.

---

## D. Artifact persistence (OpenSpec filesystem)

Every phase that produces an artifact MUST persist it.
Skipping this BREAKS the pipeline — downstream phases will not find your output.

Write the complete artifact to its OpenSpec file under
`{store-root}/openspec/changes/{change-name}/`. Do not save SDD artifacts to Engram.

### Artifact types

| Artifact | OpenSpec path | Produced by |
|----------|---------------|-------------|
| proposal | `proposal.md` | sdd-propose |
| explore | `explore.md` | sdd-explore |
| spec | `specs/{subproject}/{domain}/spec.md` | sdd-spec |
| design | `design.md` | sdd-design |
| tasks | `tasks.md` | sdd-tasks |
| apply-progress | `apply-progress.md` | implementer |
| verify-report | `verify-report.md` | sdd-verify |
| archive-report | `archive-report.md` | sdd-archive |
| state | `state.yaml` | orchestrator |

### Delta spec format

Delta specs live at `{store-root}/openspec/changes/{change-name}/specs/{a}/{b}/spec.md`
— never a flat `specs/spec.md` (validate rejects it and it never merges). Depth rule:

- Exactly `specs/{subproject}/{domain}/spec.md`. The store is the product; never repeat the product name.
- A spec that serves several subprojects of the same product lives in the `shared/` subproject:
  `specs/shared/{domain}/spec.md`. A change touching several subprojects still carries one delta per subproject.

Reuse an existing first-level folder from `openspec list --specs --json --store <store-id>` before
creating a new one. Spec ids are `a/b`; read one with `openspec show <a/b> --type spec --store <store-id>`.
On archive, ADDED creates `openspec/specs/{a}/{b}/spec.md` in the store; MODIFIED updates that file in place.
One change may carry deltas for several subprojects; each archives into its own living spec.

Duplicate check (`sdd-spec`): nothing in openspec detects duplicate capabilities at different depths.
Before any ADDED spec, run `openspec list --specs --json --store <store-id>`. If a living spec with
the same domain exists at another path, write a MODIFIED delta against it instead, or return
`status: blocked` if unclear.

Cross-product work: two subprojects of one product = one change with one delta per subproject.
Two products = paired changes with the SAME change name in each store; each proposal names the other,
the stores reference each other via `references:` in the store's own `openspec/config.yaml`, and the
tracker epic groups them. Each validates and archives on its own. Read a referenced store's specs with
`openspec show <spec-id> --type spec --store <other-store-id>`. Do not use `initiative:` in `.openspec.yaml`.

Validation command: `openspec validate "<change>" --store <store-id> --strict --json`. Always name
the change; bare `validate` exits 1 in non-TTY.

- Section headers: `## ADDED Requirements`, `## MODIFIED Requirements`,
  `## REMOVED Requirements`, `## RENAMED Requirements`
- `### Requirement: <name>` — body must contain SHALL or MUST (`--strict` enforces this)
- `#### Scenario: <name>` — exactly 4 hashes; bullets `- **WHEN** ...`, `- **THEN** ...`,
  optional `- **AND** ...`; at least 1 scenario per requirement
- A new capability delta opens with `## Purpose` (at least 50 chars)
- MODIFIED repeats the full requirement block under a header matching the existing one
- REMOVED needs `**Reason**` + `**Migration**`; RENAMED uses `FROM:` / `TO:`

`openspec status` `isComplete` only means the artifacts exist. Tasks are done only when every task in
`tasks.md` is `- [x]`. Never use status as proof of completion.

### Conflicting active changes

Before writing a MODIFIED/REMOVED/RENAMED delta, `sdd-spec` lists active changes
(`openspec list --json --store <store-id>`) and Globs their `specs/` folders. If another active change
carries a delta for the same spec id and requirement, report it under `risks` with the change names
and record it in this change's `proposal.md` under `## Spec conflicts`, one line per partner:
`- <change-name>: <spec-id> (open)`. For paired multi-product changes, do the same in each referenced
store (`--store <other>`).

The orchestrator shows the conflict to the user, who picks the archive order. The change archived second
re-runs `sdd-spec` after the partner is archived, rewrites its delta against the updated living spec, and
marks the line `(resolved: synced after <change-name> archived)`.

`sdd-archive` enforces this: for every change in `changes/archive/` whose folder date is on or after this
change's `created` date and whose specs touch a spec id this change also touches, that partner must be
listed as `resolved` under `## Spec conflicts`; otherwise it returns `status: blocked` ("re-run sdd-spec
against the updated living spec"). An `(open)` line whose partner is still active does not block: this
change archives first and the partner must re-sync.

A dead or superseded change archives via `MODE: superseded` (see `sdd-archive.md`).

---

## E. Return envelope

> **CRITICAL — Response ordering**: write OpenSpec artifacts BEFORE your
> final text response. Your last output MUST be text, NOT a tool call.
> If you end with a tool call, the orchestrator receives only the tool result —
> your analysis is lost.

Every phase MUST return this structured envelope:

```
status: done | blocked | partial
executive_summary: 1-3 sentence summary of what was done
artifacts: list of artifact keys or paths written
next_recommended: next SDD phase to run, or "none"
risks: risks discovered, or "none"
```

Do not save SDD artifacts to Engram. The orchestrator handles non-SDD session summaries
under the global memory protocol.

---

## F. Necessary code and right-sized artifacts

Necessary code: the change produces only the code it needs, held to the quality standards in CLAUDE.md section 3 (KISS, DRY, YAGNI, SOLID, clean, scalable, maintainable). Tasks group work into units, each with a clear finish and a verification.

Right-size artifacts: artifact size follows the change, never a template. A change that needs
1 requirement gets 1 requirement. No minimum lengths, no restating living specs, no requirements
invented for completeness. Explore and design stay short when the scope is already clear; design
records only decisions that need one. Large, delicate, or foundational work (new project,
architecture) still gets the full depth.

---

## G. Living design

`design.md` stays current during implementation. When implementation proves a design decision wrong,
the implementing agent (builder/implementer) reports it as `design deviation` in its report; the
orchestrator delegates `sdd-design` right away for a small update before more implementation.
`sdd-verify` still runs once at the end, also for long changes, and flags any unrecorded deviation.

---

## H. Persistence

SDD always reads and writes OpenSpec filesystem artifacts under
`{store-root}/openspec/changes/{change-name}/`. Git history provides the audit trail.
openspec never stages or commits; after an SDD block that wrote artifacts the orchestrator
delegates `git-ops` to commit inside the store root. No SDD agent commits.
Engram is not an SDD artifact store and has no SDD persistence mode or fallback.
