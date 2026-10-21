#!/usr/bin/env python3
"""PreToolUse guard for read-only/restricted subagents' Bash calls.

Applies a per-agent_type command allow-list to tool_name == "Bash" only.
Any other agent_type (including the main thread, agent_type absent) is
passed through untouched, except that commit-creating and staging git
operations (incl. rm, mv, non-hard reset) are denied for every agent_type
but git-ops. Push is denied for everyone but git-ops (plain form only; force
push is confirmed by native `permissions.ask`); git-ops also may not run an
interactive rebase or `git restore`. Worktree mutations are denied for everyone
but git-ops; git-ops deletions (`git worktree remove|prune`, `git branch
-d|-D|--delete`) pass only in plain form, for native `permissions.ask`.

The hook never returns `ask`: a hook `ask` inside a background subagent never
reaches the user. Confirmations live in native `permissions.ask`. Discards and
stash pass through to those rules in plain `git <sub>` form; any other form
is denied because the native prefix rules cannot see it. `native_ask` commands
also pass through to native rules.
"""
import fnmatch
import json
import os
import re
import shlex
import sys

COMMIT_AGENT = "git-ops"
STAGING_GIT = {"add", "commit", "cherry-pick", "merge", "rebase", "revert", "am", "rm", "mv", "pull"}
WORKTREE_MUTATIONS = {"add", "remove", "prune", "move", "lock", "unlock", "repair"}
WORKTREE_DELETIONS = {"remove", "prune"}
BRANCH_DELETE_FLAG = re.compile(r"--delete|-[a-zA-Z]*[dD][a-zA-Z]*")
PUSH_RISKY_SHORT_CLUSTER = re.compile(r"-[a-zA-Z]{2,}")
GIT_OPTIONS_WITH_VALUE = {"-C", "-c", "--git-dir", "--work-tree", "--namespace"}
SEGMENT_SEPARATORS = re.compile(r"\$\(|[;&|`\n]")

GIT_ALLOW = [
    "git status*",
    "git log*",
    "git diff*",
    "git show*",
    "git branch --list*",
    "git worktree list*",
    "git blame*",
]

VALIDATION_DENY = [
    "npm run *fix*", "pnpm run *fix*", "yarn run *fix*", "bun run *fix*",
    "eslint *--fix*", "ruff format*", "ruff check *--fix*", "cargo clippy *--fix*",
    "uv run ruff format*", "uv run ruff check *--fix*",
    "poetry run ruff format*", "poetry run ruff check *--fix*",
    "rubocop *-A*", "rubocop *-a*", "rubocop *--autocorrect*",
    "bundle exec rubocop *-A*", "bundle exec rubocop *-a*", "bundle exec rubocop *--autocorrect*",
]

VALIDATION_ALLOW = [
    # Node.js
    "npm test*", "npm run test*", "npm run lint*", "npm run typecheck*", "npm run format-check*", "npm run format:check*", "npm run build*",
    "pnpm test*", "pnpm run test*", "pnpm run lint*", "pnpm run typecheck*", "pnpm run format-check*", "pnpm run format:check*", "pnpm run build*",
    "yarn test*", "yarn run test*", "yarn run lint*", "yarn run typecheck*", "yarn run format-check*", "yarn run format:check*", "yarn run build*",
    "bun test*", "bun run test*", "bun run lint*", "bun run typecheck*", "bun run format-check*", "bun run format:check*", "bun run build*",
    "tsc", "tsc *", "prettier --check*", "eslint*",
    # Python
    "uv run pytest*", "uv run ruff check*", "uv run mypy*", "uv run pyright*",
    "pytest*", "python -m pytest*", "ruff check*", "mypy*", "pyright*", "flake8*", "pip check*",
    "poetry run pytest*", "poetry run ruff check*", "poetry run mypy*", "poetry run pyright*", "poetry run flake8*", "poetry run python -m compileall*",
    "python -m compileall*",
    # Java / Kotlin
    "mvn test*", "mvn verify*", "mvn compile*", "mvn checkstyle*",
    "gradle test*", "gradle check*", "gradle compile*", "gradle lint*",
    # Go
    "go test*", "go vet*", "go build*", "gofmt -d*", "staticcheck*",
    # Rust
    "cargo test*", "cargo check*", "cargo clippy*", "cargo fmt --check*", "cargo build*",
    # .NET
    "dotnet test*", "dotnet build*", "dotnet format --verify-no-changes*",
    # Ruby
    "bundle exec rspec*", "bundle exec rubocop*", "rake test*", "ruby -c*", "rubocop*",
    # PHP
    "composer test*", "phpunit*", "phpstan*", "phpcs*", "php -l*",
    # Generic project checks
    "make test*", "make lint*", "make check*", "make format-check*", "make typecheck*", "make compile*", "make build*",
]

SHELL_METACHARACTERS = ["&", ";", "|", "$(", "`", "\n", ">", "<"]

RULES = {
    "sdd-explore": {
        "allow": ["openspec context*", "openspec doctor*", "openspec list*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-propose": {
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-spec": {
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-design": {
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-tasks": {
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-archive": {
        "allow": ["openspec status*", "openspec validate*", "openspec archive*", "openspec show*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "aws": {
        "allow": [
            "aws logs describe-*",
            "aws logs filter-log-events *",
            "aws logs get-log-events *",
            "aws logs start-query *",
            "aws logs get-query-results *",
            "aws dynamodb describe-*",
            "aws dynamodb list-*",
            "aws dynamodb get-item *",
            "aws dynamodb query *",
            "aws dynamodb scan *",
            "aws lambda get-*",
            "aws lambda list-*",
            "aws ecs describe-*",
            "aws ecs list-*",
            "aws ec2 describe-*",
        ] + GIT_ALLOW,
    },
    "codegraph-maintainer": {
        "native_ask": [
            "codegraph unlock*",
            "codegraph init*",
            "codegraph sync*",
            "codegraph index*",
            "codegraph-health*",
            "scripts/codegraph-health.sh*",
            "./scripts/codegraph-health.sh*",
        ],
        "allow": [
            "codegraph status*",
            "codegraph query*",
            "codegraph explore*",
            "codegraph files*",
            "codegraph node*",
            "codegraph callers*",
            "codegraph callees*",
            "codegraph impact*",
            "codegraph affected*",
        ] + GIT_ALLOW,
    },
    "engram-maintainer": {
        "native_ask": ["engram delete *", "engram export *"],
        "allow": [
            "engram context*",
            "engram search*",
            "engram stats*",
            "engram projects list*",
            "engram doctor*",
            "engram timeline*",
        ],
    },
    "code-reviewer": {
        "allow": VALIDATION_ALLOW + GIT_ALLOW,
    },
    "debugger": {
        "allow": VALIDATION_ALLOW + GIT_ALLOW + [
            "rg *",
            "wc *",
            "du *",
            "ls *",
            "cat *",
            "head *",
            "tail *",
            "zcat *",
            "gzip -cd *",
            "journalctl *",
            "docker logs *",
            "kubectl logs *",
        ],
    },
}

RULES["sdd-verify"] = {
    "allow": RULES["code-reviewer"]["allow"] + ["openspec status*", "openspec validate*", "openspec store list*"],
}
RULES["judge-a"] = RULES["code-reviewer"]
RULES["judge-b"] = RULES["code-reviewer"]


def git_invocations(command: str):
    """Return (subcommand, args) for every git call found in `command`.

    Splits on chain separators, finds a token whose basename is `git` (so
    `/usr/bin/git` and `env X=1 git` work), then skips global options such as
    `-C <path>`, `-c k=v`, `--git-dir=...`, `--no-pager`. `commit-tree` is its
    own subcommand, so it never matches `commit`.

    shortcut: `sh -c "git ..."` strings, aliases and variable-built commands
    can evade; quoted text that merely mentions `git commit` fails closed.
    Upgrade to a real shell parser if that matters.
    """
    calls = []
    for segment in SEGMENT_SEPARATORS.split(command):
        try:
            tokens = shlex.split(segment)
        except ValueError:
            tokens = segment.split()
        for index, token in enumerate(tokens):
            if os.path.basename(token) != "git":
                continue
            rest = iter(tokens[index + 1:])
            for option in rest:
                if option in GIT_OPTIONS_WITH_VALUE:
                    next(rest, None)
                elif not option.startswith("-"):
                    calls.append((option, list(rest)))
                    break
            break
    return calls


def is_staging_git(subcommand: str, args) -> bool:
    return subcommand in STAGING_GIT or (subcommand == "reset" and "--hard" not in args)


def is_plain_git_start(command: str, subcommand: str) -> bool:
    return re.match(rf"git\s+{re.escape(subcommand)}(\s|$)", command.lstrip()) is not None


def is_deleting_git(subcommand: str, args) -> bool:
    if subcommand == "worktree":
        return bool(args) and args[0] in WORKTREE_DELETIONS
    return subcommand == "branch" and any(BRANCH_DELETE_FLAG.fullmatch(a) for a in args)


def has_force_or_delete_cluster(push_args) -> bool:
    """True for letter clusters like `-uf`, which evade the native `-f`/`-d` ask globs."""
    options = push_args[: push_args.index("--")] if "--" in push_args else push_args
    return any(
        PUSH_RISKY_SHORT_CLUSTER.fullmatch(a) and ("f" in a or "d" in a) for a in options
    )


def has_delete_refspec(push_args) -> bool:
    """True for `:branch` refspecs; `:*` is legacy prefix syntax, so no native ask glob can match them."""
    return any(a.startswith(":") for a in push_args)


def discard_verdict(command: str, is_first_call: bool, subcommand: str, args):
    if subcommand == "rebase" and any(a in ("-i", "--interactive") for a in args):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{COMMIT_AGENT}' must not run "
            f"an interactive rebase; return status: blocked instead."
        )
    needs_native_confirmation = (
        subcommand == "stash"
        or subcommand == "clean"
        or (subcommand == "reset" and "--hard" in args)
        or (subcommand == "checkout" and "--" in args)
        or subcommand == "restore"
    )
    if needs_native_confirmation and not (is_first_call and is_plain_git_start(command, subcommand)):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: `git {subcommand}` needs native user "
            f"confirmation, which only triggers for the plain `git {subcommand} ...` form "
            f"as the whole command; rerun it that way."
        )
    return None


def matches_any(command: str, patterns) -> bool:
    return any(fnmatch.fnmatchcase(command, p) for p in patterns)


def has_shell_metacharacters(command: str) -> bool:
    return any(token in command for token in SHELL_METACHARACTERS)


def decide(agent_type: str, command: str):
    calls = git_invocations(command)
    if any(subcommand == "push" for subcommand, _ in calls):
        if agent_type != COMMIT_AGENT:
            return "deny", (
                f"Blocked by infra-agent-bash-guard: only '{COMMIT_AGENT}' may run `git push`."
            )
        if not (len(calls) == 1 and is_plain_git_start(command, "push")):
            return "deny", (
                "Blocked by infra-agent-bash-guard: `git push` must be the plain "
                "`git push ...` form as the whole command, so native force-push "
                "confirmation can match it."
            )
        push_args = next(args for sub, args in calls if sub == "push")
        if has_force_or_delete_cluster(push_args):
            return "deny", (
                "Blocked by infra-agent-bash-guard: use separate flags (e.g. `-u -f`) so "
                "native confirmation fires."
            )
        if has_delete_refspec(push_args):
            return "deny", (
                "Blocked by infra-agent-bash-guard: use `git push <remote> --delete <branch>` "
                "so native confirmation fires."
            )
        return None
    if agent_type == COMMIT_AGENT:
        deleting = [sub for sub, args in calls if is_deleting_git(sub, args)]
        if deleting and not (len(calls) == 1 and is_plain_git_start(command, deleting[0])):
            return "deny", (
                "Blocked by infra-agent-bash-guard: worktree/branch deletion must be the plain "
                "`git worktree remove|prune ...` or `git branch -d|-D ...` form as the whole "
                "command, so native confirmation can match it."
            )
    if agent_type == COMMIT_AGENT and any(subcommand == "restore" for subcommand, _ in calls):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{COMMIT_AGENT}' unstages with "
            f"`git reset -q -- <file>`; it never runs git restore."
        )
    if agent_type != COMMIT_AGENT and any(is_staging_git(sub, args) for sub, args in calls):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: only '{COMMIT_AGENT}' may run "
            f"commit-creating or staging git operations."
        )
    if agent_type != COMMIT_AGENT and any(
        sub == "worktree" and args and args[0] in WORKTREE_MUTATIONS for sub, args in calls
    ):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: only '{COMMIT_AGENT}' may create, "
            f"change or remove git worktrees."
        )

    rules = RULES.get(agent_type)
    if rules is None:
        for index, (subcommand, args) in enumerate(calls):
            verdict = discard_verdict(command, index == 0, subcommand, args)
            if verdict:
                return verdict
        return None

    if has_shell_metacharacters(command):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{agent_type}' agent is "
            f"read-only and command chaining/substitution characters "
            f"(&&, ;, |, $(), backticks, newlines, redirects) are not "
            f"permitted, regardless of allow-list matches."
        )

    if matches_any(command, VALIDATION_DENY):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{agent_type}' agent is "
            f"read-only; this command is on its explicit deny list "
            f"(an auto-fix/format command)."
        )
    if matches_any(command, rules.get("native_ask", [])):
        return None
    if matches_any(command, rules["allow"]):
        return "allow", (
            f"infra-agent-bash-guard: command matches '{agent_type}' "
            f"read-only allow-list."
        )
    return "deny", (
        f"Blocked by infra-agent-bash-guard: '{agent_type}' agent is "
        f"restricted to a read-only command allow-list; "
        f"this command is not on it."
    )


def evaluate_payload(raw: str):
    try:
        payload = json.loads(raw)
    except Exception:
        return "ask", "infra-agent-bash-guard: could not parse hook payload"

    if payload.get("tool_name") != "Bash":
        return None
    command = (payload.get("tool_input") or {}).get("command", "")
    return decide(payload.get("agent_type"), command)


def main() -> int:
    result = evaluate_payload(sys.stdin.read())

    if result is None:
        return 0
    decision, reason = result

    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": decision,
            "permissionDecisionReason": reason,
        }
    }))
    return 0


def _selftest() -> int:
    cases = [
        ("aws", "aws logs describe-log-groups && git push origin main", "deny"),
        ("codegraph-maintainer", "codegraph status && git commit -m pwned", "deny"),
        ("engram-maintainer", "engram context", "allow"),
        ("engram-maintainer", "engram delete 42", None),
        ("engram-maintainer", "engram delete project demo --hard", None),
        ("engram-maintainer", "engram export backup.json", None),
        ("codegraph-maintainer", "codegraph sync", None),
        ("engram-maintainer", "engram save title text", "deny"),
        ("aws", "aws logs describe-log-groups", "allow"),
        ("code-reviewer", "npm test", "allow"),
        ("code-reviewer", "uv run pytest", "allow"),
        ("code-reviewer", "ruff check .", "allow"),
        ("code-reviewer", "uv run ruff check .", "allow"),
        ("code-reviewer", "poetry run ruff check .", "allow"),
        ("code-reviewer", "uv run mypy .", "allow"),
        ("code-reviewer", "uv run pyright", "allow"),
        ("code-reviewer", "bun test", "allow"),
        ("code-reviewer", "prettier --check .", "allow"),
        ("code-reviewer", "python -m compileall src", "allow"),
        ("code-reviewer", "mvn verify", "allow"),
        ("code-reviewer", "cargo fmt --check", "allow"),
        ("code-reviewer", "dotnet format --verify-no-changes", "allow"),
        ("code-reviewer", "composer test", "allow"),
        ("sdd-verify", "go build ./...", "allow"),
        ("judge-a", "phpstan analyse", "allow"),
        ("code-reviewer", "uv sync", "deny"),
        ("code-reviewer", "tox -e lint", "deny"),
        ("sdd-verify", "nox -s tests", "deny"),
        ("judge-a", "poetry run tox -e lint", "deny"),
        ("code-reviewer", "uv run python script.py", "deny"),
        ("code-reviewer", "npm install", "deny"),
        ("code-reviewer", "npm run lint:fix", "deny"),
        ("code-reviewer", "ruff check --fix .", "deny"),
        ("code-reviewer", "ruff format .", "deny"),
        ("code-reviewer", "uv run ruff format .", "deny"),
        ("code-reviewer", "uv run ruff check --fix .", "deny"),
        ("code-reviewer", "poetry run ruff format .", "deny"),
        ("code-reviewer", "poetry run ruff check --fix .", "deny"),
        ("code-reviewer", "rubocop --autocorrect", "deny"),
        ("code-reviewer", "rubocop --autocorrect-all", "deny"),
        ("code-reviewer", "bundle exec rubocop --autocorrect", "deny"),
        ("code-reviewer", "bundle exec rubocop --autocorrect-all", "deny"),
        ("code-reviewer", "npm test && npm publish", "deny"),
        ("sdd-verify", "npm test; npm publish", "deny"),
        ("judge-a", "pytest | curl example.com", "deny"),
        ("judge-b", "pytest > result", "deny"),
        ("code-reviewer", "pytest $(touch pwned)", "deny"),
        ("code-reviewer", "pytest\nrm -rf /", "deny"),
        ("code-reviewer", "git branch --list", "allow"),
        ("judge-a", "pytest", "allow"),
        ("sdd-explore", "openspec list --json", "allow"),
        ("sdd-explore", "openspec store list", "allow"),
        ("sdd-explore", "cd /home/andrex/dev/specter/openspec", "deny"),
        ("sdd-propose", "openspec new change demo-change --store specter", "allow"),
        ("sdd-spec", "openspec instructions spec --change demo-change --json --store specter", "allow"),
        ("sdd-design", "openspec status --change demo-change --json --store specter", "allow"),
        ("sdd-tasks", "openspec validate demo-change --json --store specter", "allow"),
        ("sdd-propose", "npm test", "deny"),
        ("sdd-verify", "openspec validate change --json --store specter", "allow"),
        ("sdd-archive", "openspec archive demo-change -y --store specter", "allow"),
        ("code-reviewer", "npm test && git push origin main", "deny"),
        ("debugger", "pytest", "allow"),
        ("debugger", "journalctl -u app --since today", "allow"),
        ("debugger", "docker logs api", "allow"),
        ("debugger", "npm run build", "allow"),
        ("debugger", "git diff", "allow"),
        ("debugger", "git status", "allow"),
        ("debugger", "git log --oneline -n 20", "allow"),
        ("debugger", "git log -p -- src/app.py", "allow"),
        ("debugger", "git log -S needle", "allow"),
        ("debugger", "git show HEAD~1", "allow"),
        ("debugger", "git show HEAD~1:src/app.py", "allow"),
        ("debugger", "git blame -L 10,20 src/app.py", "allow"),
        ("debugger", "git diff main...HEAD", "allow"),
        ("debugger", "git add .", "deny"),
        ("debugger", "tail app.log", "allow"),
        ("debugger", "rm -rf dist", "deny"),
        ("debugger", "sed -i s/a/b/ f", "deny"),
        ("debugger", "tail app.log > out.txt", "deny"),
        ("debugger", "ruff check --fix .", "deny"),
        ("debugger", "git commit -m x", "deny"),
        ("debugger", "git reset --hard", "deny"),
        ("code-reviewer", "git add .", "deny"),
        ("code-reviewer", "git rm f", "deny"),
        ("code-reviewer", "git restore --staged f", "deny"),
        ("code-reviewer", "git stash", "deny"),
        ("code-reviewer", "git clean -fd", "deny"),
        ("code-reviewer", "git checkout -- f", "deny"),
        ("code-reviewer", "git restore f", "deny"),
        ("git-ops", "git commit -m 'feat: x'", None),
        ("git-ops", "git add a.py b.py", None),
        ("builder", "git commit -m x", "deny"),
        ("builder", "git pull", "deny"),
        ("git-ops", "git pull", None),
        ("builder", "cd x && git commit -m x", "deny"),
        (None, "git add .", "deny"),
        (None, "cd x && git add -A", "deny"),
        (None, "git status", None),
        ("builder", "npm test", None),
        ("builder", "git -C x commit -m y", "deny"),
        ("builder", "git -c a=b commit", "deny"),
        ("builder", "/usr/bin/git add .", "deny"),
        ("builder", "env git commit", "deny"),
        ("builder", "git --no-pager commit", "deny"),
        ("builder", "git cherry-pick abc", "deny"),
        ("builder", "git merge feat", "deny"),
        ("builder", "git rebase main", "deny"),
        ("builder", "git commit-tree x", None),
        ("builder", "git status", None),
        ("builder", "git log", None),
        ("builder", "git -C x status", None),
        ("git-ops", "git commit -m x", None),
        ("git-ops", "git -C x push", "deny"),
        ("git-ops", "git push origin main", None),
        ("git-ops", "git push --force origin main", None),
        ("git-ops", "git commit -m x && git push", "deny"),
        ("git-ops", "/usr/bin/git push", "deny"),
        ("builder", "git push origin main", "deny"),
        (None, "git push origin main", "deny"),
        ("builder", "git -C x push", "deny"),
        (None, "/usr/bin/git push", "deny"),
        ("builder", "git status & git commit -m x", "deny"),
        ("builder", "git reset --hard", None),
        ("builder", "git -C x reset --hard", "deny"),
        (None, "git restore a.json", None),
        (None, "cd x && git restore a.json", "deny"),
        ("builder", "/usr/bin/git clean -f", "deny"),
        ("builder", "env git restore f", "deny"),
        ("builder", "git status && git reset --hard", "deny"),
        ("git-ops", "git clean -f", None),
        ("git-ops", "git reset --hard", None),
        ("git-ops", "git rebase -i main", "deny"),
        ("git-ops", "git stash", None),
        ("git-ops", "git rebase main", None),
        ("git-ops", "git restore --staged a.json", "deny"),
        ("git-ops", "git rm old.md", None),
        ("git-ops", "git mv a b", None),
        ("git-ops", "git restore a.json", "deny"),
        ("git-ops", "git reset -q -- a.json", None),
        ("git-ops", "git reset --soft HEAD~1", None),
        ("builder", "git restore --staged a.json", None),
        ("builder", "git restore a.json", None),
        ("builder", "git reset a.json", "deny"),
        ("builder", "git reset --soft HEAD~1", "deny"),
        ("builder", "git rm x", "deny"),
        ("builder", "git mv a b", "deny"),
        (None, "git restore --staged a.json", None),
        (None, "git rm x", "deny"),
        (None, "git mv a b", "deny"),
        ("git-ops", "git worktree add -b fix/x ../repo-fix-x main", None),
        ("git-ops", "git worktree remove ../repo-fix-x", None),
        ("git-ops", "git -C . worktree remove ../x", "deny"),
        ("git-ops", "git status && git branch -D x", "deny"),
        ("git-ops", "git branch -D x", None),
        ("git-ops", "env git worktree prune", "deny"),
        (None, "git worktree add -b fix/x ../repo-fix-x main", "deny"),
        ("builder", "git worktree add -b fix/x ../repo-fix-x main", "deny"),
        ("builder", "git worktree remove ../repo-fix-x", "deny"),
        (None, "git worktree list", None),
        ("code-reviewer", "git worktree list", "allow"),
        ("git-ops", "git push -uf origin x", "deny"),
        ("git-ops", "git push origin -ud x", "deny"),
        ("git-ops", "git push -u origin x", None),
        ("git-ops", "git push -f origin x", None),
        ("git-ops", "git push --follow-tags", None),
        ("git-ops", "git push -n", None),
        ("git-ops", "git push origin HEAD:main", None),
        ("git-ops", "git push origin a:b", None),
        ("git-ops", "git push origin :x", "deny"),
        ("git-ops", "git push -u origin x", None),
        ("git-ops", "git -C . branch -dr x", "deny"),
        ("git-ops", "git branch -dr x", None),
        ("git-ops", "git branch --delete x", None),
        ("git-ops", "git worktree prune", None),
        ("git-ops", "git branch -vv", None),
        ("git-ops", "git -C . branch -vv", None),
        ("git-ops", "git -C . branch --sort=-committerdate", None),
        ("builder", "git branch -D x", None),
        ("builder", "git -C . worktree list", None),
    ]
    for agent_type, command, expected in cases:
        result = decide(agent_type, command)
        decision = result[0] if result else None
        assert decision == expected, (
            f"FAIL: agent={agent_type!r} command={command!r} "
            f"expected={expected!r} got={decision!r}"
        )
        print(f"PASS: agent={agent_type!r} command={command!r} -> {decision!r}")

    unparsable = evaluate_payload("not json")
    assert unparsable and unparsable[0] == "ask", f"FAIL: unparsable payload got={unparsable!r}"
    print("PASS: unparsable payload -> 'ask'")

    print("All self-test cases passed.")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        sys.exit(_selftest())
    sys.exit(main())
