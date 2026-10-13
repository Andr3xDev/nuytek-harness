#!/usr/bin/env python3
"""PreToolUse guard for read-only/restricted subagents' Bash calls.

Applies a per-agent_type command allow-list to tool_name == "Bash" only.
Any other agent_type (including the main thread, agent_type absent) is
passed through untouched -- this hook must never affect agents outside
RULES.
"""
import fnmatch
import json
import sys

GIT_DENY = [
    "git commit*", "git push*", "git add*", "git rm*", "git mv*", "git restore --staged*",
]
GIT_ASK = [
    "git reset --hard*", "git clean -f*", "git checkout -- *", "git restore *", "git stash*",
]
GIT_ALLOW = [
    "git status*",
    "git log*",
    "git diff*",
    "git show*",
    "git branch --list*",
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

SHELL_METACHARACTERS = ["&&", ";", "|", "$(", "`", "\n", ">", "<"]

RULES = {
    "sdd-explore": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec context*", "openspec doctor*", "openspec list*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-propose": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-spec": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-design": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-tasks": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec status*", "openspec validate*", "openspec show*", "openspec new*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "sdd-archive": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": ["openspec status*", "openspec validate*", "openspec archive*", "openspec show*", "openspec instructions*", "openspec store list*"] + GIT_ALLOW,
    },
    "aws": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
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
        "deny": GIT_DENY,
        "ask": GIT_ASK + [
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
        "deny": GIT_DENY,
        "ask": ["engram delete *", "engram export *"],
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
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": VALIDATION_ALLOW + GIT_ALLOW,
    },
    "debugger": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
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
    "echor-onboarder": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": GIT_ALLOW + ["git rev-parse*", "ls *", "rg *", "cat *"],
    },
    "echor-validator": {
        "deny": GIT_DENY,
        "ask": GIT_ASK,
        "allow": GIT_ALLOW + ["git rev-parse*", "ls *", "rg *", "cat *"],
    },
}

RULES["sdd-verify"] = {
    "deny": GIT_DENY,
    "ask": GIT_ASK,
    "allow": RULES["code-reviewer"]["allow"] + ["openspec status*", "openspec validate*", "openspec store list*"],
}
RULES["judge-a"] = RULES["code-reviewer"]
RULES["judge-b"] = RULES["code-reviewer"]
RULES["echor-updater"] = RULES["echor-onboarder"]


def matches_any(command: str, patterns) -> bool:
    return any(fnmatch.fnmatchcase(command, p) for p in patterns)


def has_shell_metacharacters(command: str) -> bool:
    return any(token in command for token in SHELL_METACHARACTERS)


def decide(agent_type: str, command: str):
    rules = RULES.get(agent_type)
    if rules is None:
        return None

    if has_shell_metacharacters(command):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{agent_type}' agent is "
            f"read-only and command chaining/substitution characters "
            f"(&&, ;, |, $(), backticks, newlines, redirects) are not "
            f"permitted, regardless of allow-list matches."
        )

    if matches_any(command, rules["deny"] + VALIDATION_DENY):
        return "deny", (
            f"Blocked by infra-agent-bash-guard: '{agent_type}' agent is "
            f"read-only; this command is on its explicit deny list "
            f"(git commit, push, staging commands)."
        )
    if matches_any(command, rules["ask"]):
        return "ask", (
            f"infra-agent-bash-guard: '{agent_type}' agent requires "
            f"confirmation for this command."
        )
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


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return 0

    tool_name = payload.get("tool_name")
    agent_type = payload.get("agent_type")

    if tool_name != "Bash":
        return 0
    command = (payload.get("tool_input") or {}).get("command", "")
    result = decide(agent_type, command)

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
        ("engram-maintainer", "engram delete 42", "ask"),
        ("engram-maintainer", "engram delete project demo --hard", "ask"),
        ("engram-maintainer", "engram export backup.json", "ask"),
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
        ("echor-onboarder", "git rev-parse --short HEAD", "allow"),
        ("echor-onboarder", "rg -l lucyna /home/andrex/dev/some-repo", "allow"),
        ("echor-onboarder", "git commit -m x", "deny"),
        ("echor-updater", "ls /home/andrex/dev/echor/projects", "allow"),
        ("echor-validator", "git push origin main", "deny"),
        ("debugger", "pytest", "allow"),
        ("debugger", "journalctl -u app --since today", "allow"),
        ("debugger", "docker logs api", "allow"),
        ("debugger", "npm run build", "allow"),
        ("debugger", "git diff", "allow"),
        ("debugger", "tail app.log", "allow"),
        ("debugger", "rm -rf dist", "deny"),
        ("debugger", "sed -i s/a/b/ f", "deny"),
        ("debugger", "tail app.log > out.txt", "deny"),
        ("debugger", "ruff check --fix .", "deny"),
        ("debugger", "git commit -m x", "deny"),
        ("debugger", "git reset --hard", "ask"),
        ("code-reviewer", "git add .", "deny"),
        ("code-reviewer", "git rm f", "deny"),
        ("code-reviewer", "git restore --staged f", "deny"),
        ("code-reviewer", "git stash", "ask"),
        ("code-reviewer", "git clean -fd", "ask"),
        ("code-reviewer", "git checkout -- f", "ask"),
        ("code-reviewer", "git restore f", "ask"),
    ]
    for agent_type, command, expected in cases:
        decision, _ = decide(agent_type, command)
        assert decision == expected, (
            f"FAIL: agent={agent_type!r} command={command!r} "
            f"expected={expected!r} got={decision!r}"
        )
        print(f"PASS: agent={agent_type!r} command={command!r} -> {decision!r}")

    print("All self-test cases passed.")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        sys.exit(_selftest())
    sys.exit(main())
