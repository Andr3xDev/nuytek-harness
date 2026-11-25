---
description: Post-merge cleanup, verify the PR is merged, close the issue, remove worktree and local branch
---

Activate CLEAN MODE for: $ARGUMENTS

Running this command is the user's request for the cleanup actions below.

1. Verify ALL preconditions, read-only (`explorer` for git, `mcp__github__get_pull_request` for PR state):
   - The branch's PR is merged.
   - No uncommitted changes.
   - The branch content is already on the remote default branch.
   If any fails or is unclear: stop, change nothing, report what is missing.
2. Only when all pass:
   - Move the linked Linear issue (from `$ARGUMENTS` or the branch name) to Done, if one exists.
   - If the session is inside the worktree, leave it with `ExitWorktree` `action: "keep"`.
   - Delegate to `git-ops` (include a `TARGET REPO: <absolute path>` line when the repo/worktree is not the session directory): switch the main repo to the default branch, fast-forward it, remove the worktree (if any), delete the local branch. git-ops verifies the content is on the remote default branch first; native confirmations still apply. Never delete the remote branch unless the user asked.
3. Save the Engram session summary with `mem_session_summary`.
4. Report what was cleaned.
