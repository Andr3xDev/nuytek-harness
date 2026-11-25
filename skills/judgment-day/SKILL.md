---
name: judgment-day
description: "Trigger: judgment day, dual review, adversarial review. Run blind dual review with two judges, synthesize findings, fix confirmed issues, re-judge."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use when the user explicitly asks for judgment day, dual review, or adversarial review.

## Hard Rules

- Launch two blind judges (judge-a + judge-b) in PARALLEL with identical target and criteria.
- Never review the code yourself — the orchestrator delegates, judges execute.
- Wait for BOTH judges before synthesis — never accept a partial verdict.
- Classify warnings: normal use triggers it → `WARNING (real)`. Contrived/impossible path → `WARNING (theoretical)`.
- Ask the user before fixing Round 1 confirmed issues.
- After fixing confirmed issues, re-launch BOTH judges before declaring done. If a round leaves only theoretical warnings or suggestions, report INFO and stop; do not re-judge.
- Terminal states: `JUDGMENT: APPROVED` or `JUDGMENT: ESCALATED` only.
- After 2 fix iterations with remaining issues, ask the user whether to continue.

## Decision Gates

| Condition | Action |
|---|---|
| Target unclear | Ask for scope — do not launch judges |
| Both judges find same CRITICAL/real WARNING | Confirmed — ask user, then fix |
| One judge finds issue | Suspect — report and triage, do not auto-fix |
| Judges contradict | Escalate for manual decision |
| A round leaves only theoretical warnings/suggestions | Report as INFO and stop — do not re-judge |

## Execution Steps

1. Confirm target (files, feature, PR, architecture slice) and optional custom criteria.
2. Resolve skill paths from registry — inject same skills into both judge prompts.
3. Delegate to judge-a and judge-b concurrently with identical target and criteria.
4. Collect both results. Synthesize into the verdict table: a Markdown table, one row per finding, columns `finding | Judge A | Judge B | severity | status`; zero findings = one row `none`. Match findings across judges by file and line and keep both ids (JA-xxx, JB-xxx) in the row. Status:
   - **Confirmed**: both judges agree → fix candidate
   - **Suspect**: one judge found it → report, don't auto-fix
   - **Contradiction**: judges disagree → escalate to user
   - **INFO**: theoretical warnings or suggestions → log only
5. Present verdict table to user. Ask before fixing confirmed issues.
6. If user approves fixes: preserve the source lane — TDD work goes to
   `implementer`; non-TDD work goes to `builder`; if unknown, rerun the
   test-value gate. Constrain remediation to confirmed issues only.
7. After fixes applied: re-launch both judges in parallel.
8. Repeat until APPROVED (zero confirmed CRITICALs and real WARNINGs; leftover theoretical warnings or suggestions are reported as INFO) or ESCALATED.

## Output Contract

Return `## Judgment Day — {target}` with:
- Round number
- Verdict table (Markdown table: finding | Judge A | Judge B | severity | status)
- Confirmed / suspect / contradiction counts
- Fixes applied (if any)
- Re-judgment result (if applicable)
- Final: `JUDGMENT: APPROVED ✅` or `JUDGMENT: ESCALATED ⚠️`
