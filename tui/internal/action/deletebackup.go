package action

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanDeleteBackup needs env.FS for the symlink guards and the size in the confirmation.
// Every entry is validated on its own and one refusal fails the whole plan.
// The plan has no store: a snapshot of a snapshot is never made.
func PlanDeleteBackup(env check.Env, t target.Target, entries ...backup.Entry) (plan.Plan, error) {
	steps := make([]plan.Step, 0, len(entries))
	stamps := make([]string, 0, len(entries))
	var total int64
	for _, b := range entries {
		if err := validateDeletable(env, t, b); err != nil {
			return plan.Plan{}, err
		}
		rows := []plan.Change{{Effect: plan.EffectDelete, Path: b.Path}}
		steps = append(steps, plan.RemoveTree{Path: b.Path, Rows: rows})
		stamps = append(stamps, b.Stamp)
		total += backup.Size(env.FS, b)
	}
	if len(entries) == 1 {
		p := plan.New("Delete backup "+stamps[0], nil, steps)
		p.Confirm = fmt.Sprintf("Delete backup %s (%d bytes)?", stamps[0], total)
		return p, nil
	}
	p := plan.New(fmt.Sprintf("Delete %d backups", len(entries)), nil, steps)
	p.Confirm = fmt.Sprintf("Delete %d backups (%s; %d bytes)?", len(entries), strings.Join(stamps, ", "), total)
	return p, nil
}

func validateDeletable(env check.Env, t target.Target, b backup.Entry) error {
	if err := backup.ValidateStamp(b.Stamp); err != nil {
		return err
	}
	root := filepath.Dir(b.Path)
	if !inStores(t, env, root) {
		return fmt.Errorf("backup %q is outside the target's backup stores", b.Path)
	}
	if b.Path != filepath.Join(root, b.Stamp) {
		return fmt.Errorf("backup %q is not directly inside its store", b.Path)
	}
	for _, path := range []string{root, b.Path} {
		info, err := env.FS.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing to delete through symlink %s", path)
		}
	}
	return nil
}
