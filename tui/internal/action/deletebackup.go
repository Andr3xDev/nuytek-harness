package action

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanDeleteBackup needs env.FS for the symlink guards and the size in the confirmation.
// The plan has no store: a snapshot of a snapshot is never made.
func PlanDeleteBackup(env check.Env, t target.Target, b backup.Entry) (plan.Plan, error) {
	if err := backup.ValidateStamp(b.Stamp); err != nil {
		return plan.Plan{}, err
	}
	root := filepath.Dir(b.Path)
	if !inStores(t, env, root) {
		return plan.Plan{}, fmt.Errorf("backup %q is outside the target's backup stores", b.Path)
	}
	if b.Path != filepath.Join(root, b.Stamp) {
		return plan.Plan{}, fmt.Errorf("backup %q is not directly inside its store", b.Path)
	}
	for _, path := range []string{root, b.Path} {
		info, err := env.FS.Lstat(path)
		if err != nil {
			return plan.Plan{}, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return plan.Plan{}, fmt.Errorf("refusing to delete through symlink %s", path)
		}
	}
	rows := []plan.Change{{Effect: plan.EffectDelete, Path: b.Path}}
	p := plan.New("Delete backup "+b.Stamp, nil, []plan.Step{plan.RemoveTree{Path: b.Path, Rows: rows}})
	p.Confirm = fmt.Sprintf("Delete backup %s (%d bytes)?", b.Stamp, backup.Size(env.FS, b))
	return p, nil
}
