#!/usr/bin/env python3
"""PreToolUse guard that reserves destructive Engram MCP tools for engram-ops.

Denies the guarded tools for every agent_type except engram-ops,
including the main thread (agent_type absent). Any other tool passes through.
The hook never returns `ask`: a hook `ask` inside a background subagent never
reaches the user.
"""
import json
import sys

MAINTAINER_AGENT = "engram-ops"
ENGRAM_MCP_PREFIXES = ("mcp__engram__", "mcp__plugin_engram_engram__")
DESTRUCTIVE_TOOL_NAMES = ("mem_delete", "mem_merge_projects")
GUARDED_TOOLS = frozenset(
    prefix + name for prefix in ENGRAM_MCP_PREFIXES for name in DESTRUCTIVE_TOOL_NAMES
)


def decide(agent_type, tool_name):
    if tool_name not in GUARDED_TOOLS or agent_type == MAINTAINER_AGENT:
        return None
    return "deny", (
        f"Blocked by engram-tool-guard: '{tool_name}' is reserved for "
        f"{MAINTAINER_AGENT}; only {MAINTAINER_AGENT} via /memory may use it."
    )


def evaluate_payload(raw: str):
    try:
        payload = json.loads(raw)
    except Exception:
        return "deny", "engram-tool-guard: could not parse hook payload"
    return decide(payload.get("agent_type"), payload.get("tool_name"))


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
    cases = []
    for prefix in ENGRAM_MCP_PREFIXES:
        for name in DESTRUCTIVE_TOOL_NAMES:
            tool = prefix + name
            cases += [
                (MAINTAINER_AGENT, tool, None),
                ("builder", tool, "deny"),
                ("git-ops", tool, "deny"),
                (None, tool, "deny"),
            ]
        cases += [
            ("builder", prefix + "mem_save", None),
            ("builder", prefix + "mem_update", None),
            (None, prefix + "mem_search", None),
        ]
    cases += [("builder", "Bash", None), (None, None, None)]

    for agent_type, tool_name, expected in cases:
        result = decide(agent_type, tool_name)
        decision = result[0] if result else None
        assert decision == expected, (
            f"FAIL: agent={agent_type!r} tool={tool_name!r} "
            f"expected={expected!r} got={decision!r}"
        )
        print(f"PASS: agent={agent_type!r} tool={tool_name!r} -> {decision!r}")

    unparsable = evaluate_payload("not json")
    assert unparsable and unparsable[0] == "deny", f"FAIL: unparsable payload got={unparsable!r}"
    print("PASS: unparsable payload -> 'deny'")

    print("All self-test cases passed.")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        sys.exit(_selftest())
    sys.exit(main())
