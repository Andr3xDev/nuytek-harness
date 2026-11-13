# nuytek-harness TUI

Terminal dashboard and doctor for the harness repo. It shows, per CLI target, which manifest components are installed and which health checks pass. It also runs install, update, pull and backup actions.

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
- `--dry-run`: apply no step, write no files and save no state; read-only health checks still run. Every step is listed as not executed.

The TUI needs a terminal on both stdin and stdout. Otherwise it prints `nuytek-harness: a terminal is required` to stderr and exits with code 1, before it reads or writes any state.

## Actions

Each CLI target has these five tabs, switched with `[` and `]`, or `h` and `l` (`←`/`→`):

- Dashboard: installed components and health checks.
- Update: changes the repo has that the machine does not, then applies them.
- Pull: copies machine changes back into the repo.
- Backups: lists backups, with backup now (`b`), restore (`r`) and delete (`d`).
- Manage: read-only list of the target's MCPs, Plugins and Skills. Skills that are not in the harness carry a `local` tag.

Press `i` on the dashboard to open the install wizard. `m` switches between Install (pick components with `space`, all or none with `a`) and Hard install (resets every directory component and prunes MCP servers and plugins the repo does not declare). Each action shows a plan, goes to confirmation with `enter` and runs on `y`, snapshots to a backup first, then streams the step output. `ctrl+c` cancels a running action (`q` is refused during a run). `esc` goes back to the sidebar (or `tab`) and `?` lists the keys of the current view.

## Specter

The first sidebar entry, above the CLI targets, is the global Specter screen. It is read-only except for the actions below and shows these sections:

- Root: the resolved specter root and where it came from. When the root is missing it shows a `git clone` command as text; the TUI never runs git. Set `NUYTEK_SPECTER_REPO` to fill the repo URL into that hint.
- Stores: each registered store with its repos, Engram projects and active, archived and spec counts.
- Checks: `openspec`, registration, `repos.yaml`, and whether the harness agent files reference the root.
- Drift: report-only items, each with a `/specter` suggestion.
- Validation: per-store `openspec validate` results, kept for the session only.

Keys on this screen:

- `v`: validate all stores.
- `f`: fast init. Registers every unregistered store folder and can map the current repo to a store.
- `m`: maintenance. Registers unregistered store folders; other drift stays report-only. Elsewhere `m` is the install mode.
- `r`: refresh.

Fast init and maintenance take no snapshot and never delete anything. They need the root under `$HOME`.

`NUYTEK_SPECTER_ROOT` sets the specter root (default `~/dev/specter`). Only the TUI reads it; the scripts and agents keep using `~/dev/specter`, and a Checks row warns when the two differ.

A check that cannot run, such as one that needs a missing `openspec`, has status `skipped` (glyph `-`). It never counts as a failure.

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

Packages under `tui/internal` each own their types and logic.

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
| `specter` | Specter root, stores, checks, drift, validation and write plans |
| `targets/claude` | Claude Code composition: `Spec()` and its extractors |
| `ui` | Bubble Tea presentation |

`cmd/nuytek-harness` is the only wiring point.

Dependency rule: interfaces are declared by the consumer, and each package imports only what it needs.

- `manifest`, `check`, `backup`, `repo`, `state` import no internal package.
- `drift` imports `check`, `manifest`.
- `target` imports `backup`, `check`, `drift`, `manifest`.
- `dashboard` imports `target` and what `target` imports.
- `specter` imports `check`, `drift`, `manifest`, `plan`, `repo`. It imports no target, action, or ui.
- `osfs`, `osexec` import `check`.
- `targets/<cli>` imports `target`, `check`, `drift`, `manifest`, `backup`.
- `ui` imports `action`, `backup`, `check`, `dashboard`, `drift`, `plan`, `specter`, `target`. It never imports `targets/*`, `osfs`, `osexec`, or `state`.

## Adding a new CLI target

Example: OpenCode. Up to four files: a manifest, a Spec, a one-line swap in main.go (plus its import), and optional tests.

1. `targets/opencode.toml`: new manifest with `[[component]]` entries (`kind`, `source`, `target`, `needs`).
2. `tui/internal/targets/opencode/opencode.go`: new file with `func Spec() target.Spec`. Set `ID`, `Name`, `Evaluators` (usually `drift.Evaluators(nil)`), `Checks` composed from the generic `check` kinds, and `BackupDirs` if any. Paths come from `manifest.Manifest.TargetOf`.
3. `tui/cmd/nuytek-harness/main.go`: replace `target.Planned("opencode", "OpenCode")` in `specs` with `opencode.Spec()` and add the import.
4. `tui/internal/targets/opencode/*_test.go`: optional, only for OpenCode-specific extractors.

For the actions, also:

- Set `pull = true` on each component that `Pull` should copy back into the repo.
- Return two unique entries from `BackupDirs`: one under home and one under the repo.
- Add `Special` kind operations (`map[manifest.Kind]plan.KindOps`) only for kinds that are not plain files or directories.
- Add a `.gitignore` pattern for the repo backup store, like the existing `backups/`.

No edit to `manifest`, `check`, `drift`, `backup`, `target`, `plan`, `action`, `writer`, `dryrun`, `dashboard`, or `ui`: the sidebar and dashboard render whatever targets they receive, and the actions work from the `Spec`. If OpenCode needs a new check kind, write it inside `targets/opencode` first and promote it to `check` only when a second target needs it.
