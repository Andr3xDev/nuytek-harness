---
description: Manage the Echor vault — onboard, update, close decisions, ask, or validate project knowledge
---

Activate ECHOR MODE for: $ARGUMENTS

Subcommands:
1. `onboard <repo-path>` → delegate to echor-onboarder with the repo path
2. `update <slug|all>` → delegate to echor-updater with the target slug or "all"
3. `close <slug> <decision summary>` → delegate to echor-updater to append the supplied closed decision
4. `ask <question>` → delegate to echor-consultador with the question, read-only
5. `validate [slug|all]` → delegate to echor-validator with the target slug or "all", read-only audit

If "$ARGUMENTS" has no recognized subcommand, infer the intent from the phrasing (a repo path suggests onboard, a question suggests ask, "check"/"audit"/"drift" suggests validate, "refresh"/"sync" suggests update) and state which agent was chosen before delegating.

Only onboard, update, and close write to the vault; ask and validate never modify it.
