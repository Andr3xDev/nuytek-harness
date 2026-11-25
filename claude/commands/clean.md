---
description: Post-merge cleanup, verify the PR is merged, close the issue, remove worktree and local branch
---

Activate CLEAN MODE for: $ARGUMENTS

Running this command is the user's request for the cleanup actions below.

1. Verify ALL preconditions, read-only:
   - PR: `mcp__github__list_pull_requests` with `head` = `<owner>:<branch>` and state all/closed. Owner and repo come from the `origin` URL in the main repo's `.git/config` (orchestrator Read; for a worktree, follow the `gitdir:` to the main repo); branch from the current branch (`explorer`) or `$ARGUMENTS`. Any unclear (e.g. fork PR): stop and report. Require exactly one PR with `merged_at` set. None, several, or not merged: stop and report.
   - `explorer`: no uncommitted changes in the worktree (if any) and in the main repo (`git -C <absolute path> status` for each).
   If any fails or is unclear: stop, change nothing, report what is missing.
2. If the session is inside the worktree, leave it with `ExitWorktree` `action: "keep"`.
3. Delegate to `git-ops` with `TARGET REPO: <absolute main repo path>`, the worktree path, and the statement that the user asked for worktree and local-branch removal by running /clean: switch the main repo to the default branch, fast-forward it, remove the worktree (if any), delete the local branch. No native prompt fires for worktree removal or local branch delete; the guards are the step-1 preconditions and git-ops' fetch-and-verify. Tell git-ops to fetch and verify the branch content is on the remote default branch BEFORE removing the worktree or deleting the branch, and to remove nothing and block if verification fails. Never delete the remote branch unless the user asked.
4. Only if git-ops returns `done`: move the linked Linear issue (from `$ARGUMENTS` or the branch name) to Done, if one exists. If it returns `blocked`: report it and leave the issue and everything else untouched.
5. Save the Engram session summary with `mem_session_summary`.
6. Report what was cleaned.
