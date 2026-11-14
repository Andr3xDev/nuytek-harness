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

// ErrMissingSource names a component whose repo source does not exist.
type ErrMissingSource struct{ Component, Path string }

func (e ErrMissingSource) Error() string { return "missing source for " + e.Component + ": " + e.Path }

// PlanInstall also serves update.
func PlanInstall(ctx context.Context, env check.Env, t target.Target, names []string) (plan.Plan, error) {
	comps, err := Select(t, names, ScopeInstall)
	if err != nil {
		return plan.Plan{}, err
	}
	store, err := storeOf(t, env, backup.StoreHome)
	if err != nil {
		return plan.Plan{}, err
	}
	runnable, skipped := splitBlocked(env, comps)
	parts, err := partsOf(ctx, env, t, runnable, func(ops plan.KindOps) plan.PartFunc { return ops.Install })
	if err != nil {
		return plan.Plan{}, err
	}
	p := assemble("Install", store, parts)
	p.Skipped = append(skipped, p.Skipped...)
	return p, nil
}

// splitBlocked moves the components with a tool missing from PATH to the skipped list.
func splitBlocked(env check.Env, comps []manifest.Component) (runnable []manifest.Component, skipped []plan.Skip) {
	missing := MissingTools(env.Exec, manifest.Manifest{Components: comps})
	for _, c := range comps {
		var tools []string
		for _, m := range missing {
			if slices.Contains(m.Dependents, c.Name) {
				tools = append(tools, m.Tool)
			}
		}
		if len(tools) == 0 {
			runnable = append(runnable, c)
			continue
		}
		skipped = append(skipped, plan.Skip{Name: c.Name, Reason: "missing: " + strings.Join(tools, ", ")})
	}
	return runnable, skipped
}

// installPart copies only the owned files that differ and never deletes.
func installPart(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	src, dst := componentPaths(env, c)
	if err := requireSource(env.FS, c, src); err != nil {
		return plan.Part{}, err
	}
	rows, err := drift.Diff(env.FS, src, dst, drift.InstallFilter, drift.BlockedFail)
	if err != nil {
		return plan.Part{}, err
	}
	var steps []plan.Step
	if len(rows) > 0 {
		steps = append(steps, plan.Copy{Src: src, Dst: dst, Files: changePaths(rows), Rows: rows})
	}
	paths, err := chmodPaths(env.FS, c, src, dst, true)
	if err != nil {
		return plan.Part{}, err
	}
	if len(paths) > 0 {
		steps = append(steps, plan.Chmod{Paths: paths})
	}
	return plan.Part{Steps: steps}, nil
}

func requireSource(fsys check.FS, c manifest.Component, src string) error {
	_, err := fsys.Lstat(src)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrMissingSource{Component: c.Name, Path: src}
	}
	return err
}

// chmodPaths lists what a dirx or filex component needs executable; onlyLacking keeps those without the bit now.
func chmodPaths(fsys check.FS, c manifest.Component, src, dst string, onlyLacking bool) ([]string, error) {
	var paths []string
	switch c.Kind {
	case manifest.KindDirX:
		names, err := drift.Scripts(fsys, src, drift.InstallFilter)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			paths = append(paths, filepath.Join(dst, name))
		}
	case manifest.KindFileX:
		paths = []string{dst}
	}
	if !onlyLacking {
		return paths, nil
	}
	var lacking []string
	for _, p := range paths {
		if drift.LacksExec(fsys, p) {
			lacking = append(lacking, p)
		}
	}
	return lacking, nil
}
