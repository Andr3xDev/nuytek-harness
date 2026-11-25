---
description: Post-merge cleanup, verify the PR is merged, close the issue, remove worktree and local branch
---

Activate CLEAN MODE for: $ARGUMENTS

Running this command is the user's request for the cleanup actions below.

1. Verify ALL preconditions, read-only:
   - Detection: `explorer` reports the current branch, the default branch (from `git branch --list -r`, `origin/HEAD -> origin/<default>`), and `git worktree list`. The worktree path for the resolved branch comes from that list (none if absent); use it for the cleanliness check and the git-ops delegation. Inside a worktree the current branch is that worktree's branch.
   - PR check (for a branch): `mcp__github__list_pull_requests` with `head` = `<owner>:<branch>` and state all/closed. Owner and repo come from the `origin` URL in the main repo's `.git/config` (orchestrator Read; for a worktree, follow the `gitdir:` to the main repo). Any unclear (e.g. fork PR): fail. Require exactly one PR with `merged_at` set. None, several, or not merged: fail.
   - Branch (and linked issue), first match wins:
     1. The whole `$ARGUMENTS` matches `^[A-Za-z]+-\d+$` (e.g. IO-422) = issue id: Linear `get_issue`, branch = its `gitBranchName`. No issue found: stop and report.
     2. Any other `$ARGUMENTS` = branch name: that branch.
     3. No argument and the current branch is not the default branch: the current branch.
     4. No argument and on the default branch: Linear `list_issues` with assignee "me" and state In Review; a candidate whose `gitBranchName` fails the PR check is dropped. Zero candidates: report and stop. One: use it. Several: list them and ask the user which one, change nothing until answered, then continue with the chosen branch.
     After a branch is chosen by any rule, run the PR check on it once; on failure, stop and report.
   - `explorer`: no uncommitted changes in the worktree (if any) and in the main repo (`git -C <absolute path> status` for each).
   If any fails or is unclear: stop, change nothing, report what is missing.
2. If the session is inside the worktree, leave it with `ExitWorktree` `action: "keep"`.
3. Delegate to `git-ops` with `TARGET REPO: <absolute main repo path>`, the worktree path, and the statement that the user asked for worktree and local-branch removal by running /clean: switch the main repo to the default branch, fast-forward it, remove the worktree (if any), delete the local branch. /clean always ends with the main repo on the up-to-date default branch, also when it started there. No native prompt fires for worktree removal or local branch delete; the guards are the step-1 preconditions and git-ops' fetch-and-verify. Tell git-ops to fetch and verify the branch content is on the remote default branch BEFORE removing the worktree or deleting the branch, and to remove nothing and block if verification fails. Never delete the remote branch unless the user asked.
4. Only if git-ops returns `done`: move the linked Linear issue to Done: the issue from branch rule 1 or 4, else the id parsed from the branch name (Linear `gitBranchName` format, e.g. `feature/io-462-...` gives IO-462); none parsed: skip Done and say so. If it returns `blocked`: report it and leave the issue and everything else untouched.
5. Save the Engram session summary with `mem_session_summary`.
6. Report what was cleaned.
