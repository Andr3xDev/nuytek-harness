#!/usr/bin/env python3
"""SessionStart hook that flags an unclear Engram project and a missing CodeGraph index.

Engram resolves the project from cwd and has no fallback outside a repo, so the
main agent must ask the user. SessionStart can only add context, never block.
Fails open: any error prints nothing and exits 0.
"""
import json
import os
import sys

ENGRAM_CONFIG = os.path.join(".engram", "config.json")

ENGRAM_UNCLEAR = (
    "Engram project is unclear for this cwd (not inside a git repo and no .engram/config.json). "
    "Before any work, ask the user which project to use (default `general`) and follow the "
    "Target project rules in engram-protocol.md."
)
NO_CODEGRAPH = (
    "This repo has no CodeGraph index (.codegraph/). Offer to index it via the `codegraph-ops` "
    "agent; never index without the user's OK."
)


def find_repo_root(start):
    current = start
    while True:
        if os.path.exists(os.path.join(current, ".git")):
            return current
        parent = os.path.dirname(current)
        if parent == current:
            return None
        current = parent


def has_engram_config(start):
    current = start
    while True:
        if os.path.isfile(os.path.join(current, ENGRAM_CONFIG)):
            return True
        parent = os.path.dirname(current)
        if parent == current:
            return False
        current = parent


def build_notices(cwd):
    notices = []
    repo_root = find_repo_root(cwd)
    if repo_root is None and not has_engram_config(cwd):
        notices.append(ENGRAM_UNCLEAR)
    if repo_root is not None and not os.path.isdir(os.path.join(repo_root, ".codegraph")):
        notices.append(NO_CODEGRAPH)
    return notices


def main() -> int:
    try:
        raw = sys.stdin.read() if not sys.stdin.isatty() else ""
        cwd = (json.loads(raw).get("cwd") if raw.strip() else None) or os.getcwd()
        notices = build_notices(os.path.realpath(cwd))
        if notices:
            print(json.dumps({
                "hookSpecificOutput": {
                    "hookEventName": "SessionStart",
                    "additionalContext": "\n".join(notices),
                }
            }))
    except Exception:
        pass
    return 0


def _selftest() -> int:
    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        plain = os.path.join(tmp, "plain")
        repo = os.path.join(tmp, "repo")
        indexed = os.path.join(tmp, "indexed")
        configured = os.path.join(tmp, "configured")
        repo_subdir = os.path.join(repo, "src", "pkg")
        configured_subdir = os.path.join(configured, "sub", "dir")
        os.makedirs(repo_subdir)
        os.makedirs(configured_subdir)
        os.makedirs(os.path.join(repo, ".git"))
        os.makedirs(os.path.join(indexed, ".git"))
        os.makedirs(os.path.join(indexed, ".codegraph"))
        os.makedirs(os.path.join(configured, ".engram"))
        os.makedirs(plain)
        with open(os.path.join(configured, ENGRAM_CONFIG), "w") as handle:
            handle.write("{}")
        cases = [
            (plain, [ENGRAM_UNCLEAR]),
            (repo, [NO_CODEGRAPH]),
            (indexed, []),
            (configured, []),
            (repo_subdir, [NO_CODEGRAPH]),
            (configured_subdir, []),
        ]
        for cwd, expected in cases:
            got = build_notices(cwd)
            assert got == expected, f"FAIL: cwd={cwd!r} expected={expected!r} got={got!r}"
            print(f"PASS: {os.path.basename(cwd)} -> {len(got)} notice(s)")
    print("All self-test cases passed.")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        sys.exit(_selftest())
    sys.exit(main())
