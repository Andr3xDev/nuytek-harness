package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osexec"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/repo"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/targets/claude"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/ui"
)

const (
	exitFailure    = 1
	exitNoRepo     = 2
	repoEnvVar     = "NUYTEK_HARNESS_REPO"
	stateFileRel   = ".local/state/nuytek-harness/state.toml"
	flagRepoUsage  = "path to the harness repo"
	flagDryRunHelp = "apply no step, write no files and save no state; read-only health checks still run"

	terminalRequiredMsg = "nuytek-harness: a terminal is required"
)

type stateStore interface {
	Load() (state.State, string)
	Save(state.State) error
}

func main() { os.Exit(run()) }

func run() int {
	repoFlag := flag.String("repo", "", flagRepoUsage)
	dryRun := flag.Bool("dry-run", false, flagDryRunHelp)
	flag.Parse()

	if !terminalGate(term.IsTerminal) {
		fmt.Fprintln(os.Stderr, terminalRequiredMsg)
		return exitFailure
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fail(exitFailure, err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fail(exitFailure, err)
	}

	runner, store := wireIO(*dryRun, state.NewStore(filepath.Join(home, stateFileRel)), time.Now)
	saved, warning := store.Load()
	// The alt screen hides stderr, so warnings go to the TUI; they hit stderr only if it never starts.
	var warnings []string
	if warning != "" {
		warnings = append(warnings, warning)
	}

	claudeSpec := claude.Spec()
	specs := []target.Spec{claudeSpec, target.Planned("opencode", "OpenCode"), target.Planned("pi", "Pi")}
	fsys := osfs.FS{}
	res, err := repo.Resolve(*repoFlag, os.Getenv(repoEnvVar), saved.RepoPath, cwd, target.ManifestRel(claudeSpec.ID), fsys)
	if err != nil {
		for _, w := range warnings {
			fmt.Fprintln(os.Stderr, "warning:", w)
		}
		return fail(exitNoRepo, err)
	}
	if res.Warning != "" {
		warnings = append(warnings, res.Warning)
	}
	if res.Source == repo.SourceFlag {
		if err := store.Save(state.State{RepoPath: res.Path}); err != nil {
			warnings = append(warnings, fmt.Sprintf("saving state: %v", err))
		}
	}

	targets := target.Load(res.Path, specs, manifest.Load, fsys)
	env := check.Env{FS: fsys, Exec: osexec.Exec{}, LookupEnv: os.LookupEnv, Home: home, Repo: res.Path, Cwd: cwd}
	deps := ui.Deps{Env: env, Runner: runner, DryRun: *dryRun}
	if _, err := tea.NewProgram(ui.NewRoot(targets, deps, warnings)).Run(); err != nil {
		return fail(exitFailure, err)
	}
	return 0
}

func fail(code int, err error) int {
	fmt.Fprintln(os.Stderr, "error:", err)
	return code
}
