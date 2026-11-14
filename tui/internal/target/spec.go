package target

import (
	"context"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

type Spec struct {
	ID, Name   string
	Evaluators map[manifest.Kind]drift.Evaluator                                             // keys = kinds the manifest may use
	Checks     func(manifest.Manifest) ([]check.Check, error)                                // nil = planned target, not available
	BackupDirs func(home, repo, homeStore string) []backup.Dir                               // nil = none
	Special    map[manifest.Kind]plan.KindOps                                                // special kinds only; data kinds are built in
	Manage     func(ctx context.Context, env check.Env, m manifest.Manifest) []ManageSection // nil = no Manage tab

	DefaultBackupsRel string // home store, relative to HOME; "" = no configurable home store
	OwnTreeRel        string // config tree relative to HOME that a home store must stay out of
}

// HomeStore returns the saved home store, else the default under home.
func (s Spec) HomeStore(home, saved string) string {
	if saved != "" {
		return saved
	}
	return filepath.Join(home, s.DefaultBackupsRel)
}

func (s Spec) OwnTree(home string) string {
	return filepath.Join(home, s.OwnTreeRel)
}

// Stores returns the backup stores with the configured home store, nil when the target has none.
func (s Spec) Stores(env check.Env) []backup.Dir {
	if s.BackupDirs == nil {
		return nil
	}
	return s.BackupDirs(env.Home, env.Repo, s.HomeStore(env.Home, env.Saved.Targets[s.ID].BackupsDir))
}

// Planned returns a spec for a target that is announced but not implemented.
func Planned(id, name string) Spec {
	return Spec{ID: id, Name: name}
}
