package action

import (
	"context"
	"errors"
	"io/fs"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanHardInstall refuses before any step when a tool is missing, then runs every component's Hard part.
func PlanHardInstall(ctx context.Context, env check.Env, t target.Target) (plan.Plan, error) {
	if missing := MissingTools(env.Exec, t.Manifest); len(missing) > 0 {
		return plan.Plan{}, ErrMissingTools{Tools: missing}
	}
	store, err := storeOf(t, env, backup.StoreHome)
	if err != nil {
		return plan.Plan{}, err
	}
	parts, err := partsOf(ctx, env, t, t.Manifest.Components, hardOrInstall)
	if err != nil {
		return plan.Plan{}, err
	}
	p := assemble("Hard install", store, parts)
	if !p.Empty() {
		p.Confirm = confirmDeletions("Hard install", parts)
	}
	return p, nil
}

// hardDirPart rebuilds the target as the filtered source: wipe it, then copy every owned file.
func hardDirPart(ctx context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	src, dst := componentPaths(env, c)
	if err := requireSource(env.FS, c, src); err != nil {
		return plan.Part{}, err
	}
	if _, err := env.FS.Lstat(dst); errors.Is(err, fs.ErrNotExist) {
		return installPart(ctx, env, c)
	}
	// Diff diffs against the pre-wipe disk and refuses a symlink or non-directory root.
	rows, err := drift.Diff(env.FS, src, dst, drift.InstallFilter, drift.BlockedReplace)
	if err != nil {
		return plan.Part{}, err
	}
	extras, err := drift.Extras(env.FS, src, dst, drift.InstallFilter)
	if err != nil {
		return plan.Part{}, err
	}
	if len(rows) == 0 && len(extras) == 0 {
		return installPart(ctx, env, c)
	}
	deletes := make([]plan.Change, 0, len(extras))
	for _, extra := range extras {
		deletes = append(deletes, plan.Change{Effect: plan.EffectDelete, Path: extra})
	}
	owned, err := drift.OwnedFiles(env.FS, src, drift.InstallFilter)
	if err != nil {
		return plan.Part{}, err
	}
	steps := []plan.Step{
		plan.RemoveTree{Path: dst, Rows: deletes},
		plan.Copy{Src: src, Dst: dst, Files: owned, Rows: rows},
	}
	paths, err := chmodPaths(env.FS, c, src, dst, false)
	if err != nil {
		return plan.Part{}, err
	}
	if len(paths) > 0 {
		steps = append(steps, plan.Chmod{Paths: paths})
	}
	return plan.Part{Steps: steps}, nil
}
