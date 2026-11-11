package action

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanRestore copies the snapshot's files back over Base and deletes nothing; the safety snapshot goes to a store of the same kind.
func PlanRestore(_ context.Context, env check.Env, t target.Target, b backup.Entry) (plan.Plan, error) {
	if b.IsLink {
		return plan.Plan{}, fmt.Errorf("refusing to restore through symlink %s", b.Path)
	}
	info, err := env.FS.Lstat(b.Path)
	if err != nil {
		return plan.Plan{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return plan.Plan{}, fmt.Errorf("refusing to restore through symlink %s", b.Path)
	}
	store, err := storeOf(t, env, b.Kind)
	if err != nil {
		return plan.Plan{}, err
	}
	rows, err := drift.Diff(env.FS, b.Path, b.Base, drift.Filter{}, drift.BlockedFail)
	var conflict drift.ErrPathConflict
	if errors.As(err, &conflict) {
		return plan.Plan{}, fmt.Errorf("%w: restore cannot replace a directory with a file, move %s out of the way first", err, conflict.Path)
	}
	if err != nil {
		return plan.Plan{}, err
	}
	title := "Restore " + b.Stamp
	if len(rows) == 0 {
		return plan.Plan{Title: title}, nil
	}
	p := plan.New(title, store, []plan.Step{plan.Copy{Src: b.Path, Dst: b.Base, Files: changePaths(rows), Rows: rows}})
	overwrites := 0
	for _, row := range rows {
		if row.Effect == plan.EffectOverwrite {
			overwrites++
		}
	}
	p.Confirm = fmt.Sprintf("Restore %s: %d overwrite, %d create", b.Stamp, overwrites, len(rows)-overwrites)
	return p, nil
}
