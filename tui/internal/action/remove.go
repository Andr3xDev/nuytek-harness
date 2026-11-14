package action

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// PlanRemove deletes only what the repo source owns and the target holds; Remove ignores dependencies.
func PlanRemove(ctx context.Context, env check.Env, t target.Target, names []string) (plan.Plan, error) {
	comps, err := Select(t, names, ScopeRemove)
	if err != nil {
		return plan.Plan{}, err
	}
	store, err := storeOf(t, env, backup.StoreHome)
	if err != nil {
		return plan.Plan{}, err
	}
	parts, err := partsOf(ctx, env, t, comps, func(ops plan.KindOps) plan.PartFunc { return ops.Remove })
	if err != nil {
		return plan.Plan{}, err
	}
	p := assemble("Remove", store, parts)
	if !p.Empty() {
		p.Confirm = confirmDeletions("Remove", parts)
	}
	return p, nil
}

func removePart(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	src, dst := componentPaths(env, c)
	if err := requireSource(env.FS, c, src); err != nil {
		return plan.Part{}, err
	}
	dstInfo, err := env.FS.Lstat(dst)
	if errors.Is(err, fs.ErrNotExist) {
		return plan.Part{}, nil
	}
	if err != nil {
		return plan.Part{}, err
	}
	if dstInfo.IsDir() != c.Kind.IsDir() {
		return plan.Part{}, drift.ErrPathConflict{Path: dst}
	}
	if !c.Kind.IsDir() {
		return plan.Part{Steps: []plan.Step{plan.Remove{Files: []string{dst}}}}, nil
	}
	owned, err := drift.OwnedFiles(env.FS, src, drift.InstallFilter)
	if err != nil {
		return plan.Part{}, err
	}
	var files, dirs []string
	for _, rel := range owned {
		srcInfo, srcErr := env.FS.Lstat(filepath.Join(src, rel))
		deployed, dstErr := env.FS.Lstat(filepath.Join(dst, rel))
		if srcErr != nil || dstErr != nil || srcInfo.IsDir() != deployed.IsDir() {
			continue
		}
		if deployed.IsDir() {
			dirs = append(dirs, filepath.Join(dst, rel))
		} else {
			files = append(files, filepath.Join(dst, rel))
		}
	}
	if len(files) == 0 {
		return plan.Part{}, nil
	}
	depth := func(p string) int { return strings.Count(p, string(filepath.Separator)) }
	slices.SortFunc(dirs, func(a, b string) int { return depth(b) - depth(a) })
	return plan.Part{Steps: []plan.Step{plan.Remove{Files: files, Dirs: append(dirs, dst)}}}, nil
}
