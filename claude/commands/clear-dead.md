---
description: Audit dead code (unused symbols, unreachable branches, orphan files); report first, apply only after OK
---

Activate CLEAR-DEAD MODE for: $ARGUMENTS

Default scope when empty, limited to the code touching it:
- Git repo: uncommitted changes plus branch changes vs the repo's default branch. If the base is unclear, ask the user; never assume `main`.
- No git: files touched in this session plus files related to the current task.
- Still undeterminable: ask the user for the scope.

Dead code only; docs and comments are out of scope (see `/clear-comments`).

Flag:
- Unused functions, classes, methods, exports, variables, and imports.
- Unreachable branches and code after unconditional exits.
- Dead or orphan source files, and unused config tied to code.
- Symbols outside the scope whose last reference the scope removed.

Suspect: any item with possible dynamic use, reflection, public API, entry points, framework hooks, tests-only references, generated files, or string/config references (DI, routes, serialization, templates, CLI entry points).

1. Delegate to `code-reviewer` in focused mode; pass the Flag and Suspect lists above as the checklist, and have it tag suspects. It must verify no references across the whole repo (code, tests, config, non-code files) via grep or CodeGraph before flagging, not only in the scope; callers outside the scope count as use.
2. Investigate suspects before reporting: delegate to `explorer` to trace dynamic, string/config, tests-only, generated, and external uses. Each suspect ends as confirmed dead (goes with the rest) or kept, with the evidence.
3. REPORT FIRST: `file:line`, the item, evidence of no use, and the proposed action (remove, keep). Show confirmed items, and kept suspects with the reason.
4. Apply nothing until the user gives OK. A block OK approves every confirmed item; a partial OK approves only the named items. Kept items are never removed. File deletions still trigger a native `rm` confirmation per file.
5. After the OK, delegate to `quick-builder` (trivial) or `builder`.
