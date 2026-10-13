---
description: Check or maintain CodeGraph indexes for repos
agent: codegraph-maintainer
---

Check CodeGraph health for: $ARGUMENTS

Default read-only behavior:
1. Run `codegraph status` or `codegraph-health` for target repos.
2. Report which repos are indexed, missing `.codegraph/`, stale, or failed.
3. Do not run `init`, `sync`, or `index` unless explicitly requested.

If user includes `--init`, `--sync`, or `--index`, run the matching CodeGraph command only for target repos.
