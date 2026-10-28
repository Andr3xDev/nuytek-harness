---
description: Maintain OpenSpec stores, repos.yaml and spec health in specter
agent: specter-ops
context: fork
---

Run `/specter <mode> [argument]` through `specter-ops`: $ARGUMENTS

Modes:

1. `register <repo-path> [store-id]`: record the repo in `repos.yaml`; create the store only if it does not exist.
2. `setup <store-id>`: create the store in `~/dev/specter/<store-id>/` and add its `stores:` entry and README.
3. `unregister <store-id>`: unregister the store after checking that nothing still references it.
4. `references <store-id> <id...>`: set the store's `references:` in its `openspec/config.yaml`.
5. `pointer <repo-path> <store-id>`: create the repo's local, git-ignored `openspec/config.yaml` with `store: <id>`.
6. `health [store-id]`: read-only. Validate each store (`--strict`), compare specs with the code paths they describe, and report stale names. Edits nothing.

Never commits: the user commits specter by hand.
