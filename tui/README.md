# nuytek-harness TUI

Terminal dashboard and doctor for the harness repo. It shows, per CLI target, which manifest components are installed and which health checks pass. This version is read-only apart from its own state file.

## Build and install

Requires Go (see `go.mod`). From the repo root:

```sh
scripts/bootstrap-tui.sh
```

The script builds `tui/cmd/nuytek-harness` and installs it to `~/.local/bin/nuytek-harness`. It writes nothing else and does not launch the TUI. Then run:

```sh
nuytek-harness --repo <repo-root>
```

## Flags

- `--repo <path>`: path to the harness repo. When it resolves, it is saved to `~/.local/state/nuytek-harness/state.toml`.
- `--dry-run`: never write state.

## Repo discovery

The first source that holds the marker file `targets/claude.toml` wins:

1. `--repo` flag.
2. `NUYTEK_HARNESS_REPO` environment variable.
3. `repo_path` in the state file.
4. The current directory or a parent of it.

A flag or env value without the marker is an error. A state path without it is ignored with a warning. If nothing resolves, the program exits with code 2.

## Uninstall and rollback

```sh
scripts/uninstall-tui.sh
```

The script removes only `~/.local/bin/nuytek-harness` and `~/.local/state/nuytek-harness/`, then prints `removed: <path>` for each. If neither exists it prints `nothing installed`. It never touches `~/.claude`, `~/.claude.json`, backups, or the repo. It is a script only, not a TUI menu entry.

## Architecture

Packages are grouped by domain under `tui/internal`. Each owns its types and logic.

| Package | Purpose |
|---|---|
| `manifest` | Manifest model, validation, TOML loading |
| `check` | Health checks and the `FS`, `Exec`, `Env` interfaces they consume |
| `drift` | Component install state evaluators |
| `backup` | Backup directory stamps and latest lookup |
| `target` | `Spec`, `Target`, manifest path convention (`ManifestRel`) |
| `repo` | Repo discovery |
| `dashboard` | Computes the dashboard from a target |
| `state` | State file; the only package that writes files |
| `osfs`, `osexec` | OS implementations of `check.FS` and `check.Exec` |
| `targets/claude` | Claude Code composition: `Spec()` and its extractors |
| `ui` | Bubble Tea presentation |

`cmd/nuytek-harness` is the only wiring point.

Dependency rule: interfaces are declared by the consumer, and each package imports only what it needs.

- `manifest`, `check`, `backup`, `repo`, `state` import no internal package.
- `drift` imports `check`, `manifest`.
- `target` imports `backup`, `check`, `drift`, `manifest`.
- `dashboard` imports `target` and what `target` imports.
- `osfs`, `osexec` import `check`.
- `targets/<cli>` imports `target`, `check`, `drift`, `manifest`, `backup`.
- `ui` imports `dashboard`, `target`, `check`, `drift`. It never imports `targets/*`, `osfs`, `osexec`, or `state`.

## Adding a new CLI target

Example: OpenCode. Four files, one of them a single-line edit.

1. `targets/opencode.toml`: new manifest with `[[component]]` entries (`kind`, `source`, `target`, `needs`).
2. `tui/internal/targets/opencode/opencode.go`: new file with `func Spec() target.Spec`. Set `ID`, `Name`, `Evaluators` (usually `drift.Evaluators(nil)`), `Checks` composed from the generic `check` kinds, and `BackupDirs` if any. Paths come from `manifest.Manifest.TargetOf`.
3. `tui/cmd/nuytek-harness/main.go`: replace `target.Planned("opencode", "OpenCode")` in `specs` with `opencode.Spec()` and add the import.
4. `tui/internal/targets/opencode/*_test.go`: optional, only for OpenCode-specific extractors.

No edit to `manifest`, `check`, `drift`, `target`, `dashboard`, or `ui`: the sidebar and dashboard render whatever targets they receive. If OpenCode needs a new check kind, write it inside `targets/opencode` first and promote it to `check` only when a second target needs it.
