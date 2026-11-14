package action

import (
	"errors"
	"io/fs"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanBackup snapshots every installed component path into the home store.
// The Snapshot is built by hand: plan.New would derive its paths from the other steps,
// and a backup has no other step. An empty plan means nothing is installed.
func PlanBackup(env check.Env, t target.Target) (plan.Plan, error) {
	store, err := storeOf(t, env, backup.StoreHome)
	if err != nil {
		return plan.Plan{}, err
	}
	var paths []string
	for _, c := range t.Manifest.Components {
		_, dst := componentPaths(env, c)
		if _, err := env.FS.Lstat(dst); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return plan.Plan{}, err
		}
		paths = append(paths, dst)
	}
	paths = plan.DropCovered(paths)
	if len(paths) == 0 {
		return plan.Plan{Title: "Backup"}, nil
	}
	rows := make([]plan.Change, len(paths))
	for i, p := range paths {
		rows[i] = plan.Change{Effect: plan.EffectCreate, Path: p, Note: "will be saved"}
	}
	return plan.Plan{Title: "Backup", Steps: []plan.Step{plan.Snapshot{Store: *store, Paths: paths, Rows: rows}}}, nil
}
