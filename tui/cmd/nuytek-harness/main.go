package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

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
	flagDryRunHelp = "never write state"
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

	home, err := os.UserHomeDir()
	if err != nil {
		return fail(exitFailure, err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fail(exitFailure, err)
	}

	fileStore := state.NewStore(filepath.Join(home, stateFileRel))
	var store stateStore = fileStore
	if *dryRun {
		store = state.Discard(fileStore)
	}
	saved, warning := store.Load()
	if warning != "" {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}

	claudeSpec := claude.Spec()
	specs := []target.Spec{claudeSpec, target.Planned("opencode", "OpenCode"), target.Planned("pi", "Pi")}
	fsys := osfs.FS{}
	res, err := repo.Resolve(*repoFlag, os.Getenv(repoEnvVar), saved.RepoPath, cwd, target.ManifestRel(claudeSpec.ID), fsys)
	if err != nil {
		return fail(exitNoRepo, err)
	}
	if res.Warning != "" {
		fmt.Fprintln(os.Stderr, "warning:", res.Warning)
	}
	if res.Source == repo.SourceFlag {
		if err := store.Save(state.State{RepoPath: res.Path}); err != nil {
			fmt.Fprintln(os.Stderr, "warning: saving state:", err)
		}
	}

	targets := target.Load(res.Path, specs, manifest.Load, fsys)
	env := check.Env{FS: fsys, Exec: osexec.Exec{}, LookupEnv: os.LookupEnv, Home: home, Repo: res.Path}
	if _, err := tea.NewProgram(ui.NewRoot(targets, env, *dryRun)).Run(); err != nil {
		return fail(exitFailure, err)
	}
	return 0
}

func fail(code int, err error) int {
	fmt.Fprintln(os.Stderr, "error:", err)
	return code
}
