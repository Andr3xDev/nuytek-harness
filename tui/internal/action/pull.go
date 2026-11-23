package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/render"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	notPulledNote        = "not pulled: not in repo"
	notPulledSymlinkNote = "not pulled: symlink"
	notPulledStaleNote   = "not pulled: deployed copy names another Specter root; reinstall first"
)

// staleRendered reports a listed file whose repo source holds the placeholder but whose deployed copy still holds it
// or does not name the current root: unrendering it would write a wrong root into the repo.
// It mirrors `stale_listed_files` in scripts/lib.sh.
func staleRendered(fsys check.FS, repoPath, deployed, rel string, renderFn, unrenderFn plan.Transform) bool {
	if renderFn == nil || string(renderFn(rel, []byte(render.Placeholder))) == render.Placeholder {
		return false
	}
	source, err := fsys.ReadFile(filepath.Join(repoPath, rel))
	if err != nil || !bytes.Contains(source, []byte(render.Placeholder)) {
		return false
	}
	data, err := fsys.ReadFile(filepath.Join(deployed, rel))
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(render.Placeholder)) || bytes.Equal(unrenderFn(rel, data), data)
}

// ErrMissingDeployed names pull components whose deployed path is missing.
type ErrMissingDeployed struct{ Names []string }

func (e ErrMissingDeployed) Error() string {
	return "deployed path missing for: " + strings.Join(e.Names, ", ")
}

// PlanPull copies the deployed files that already exist in the repo; deployed-only files are listed as skipped.
func PlanPull(ctx context.Context, env check.Env, t target.Target, names []string) (plan.Plan, error) {
	store, err := storeOf(t, env, backup.StoreRepo)
	if err != nil {
		return plan.Plan{}, err
	}
	comps, err := Select(t, names, ScopePull)
	if err != nil {
		return plan.Plan{}, err
	}
	var missing []string
	for _, c := range comps {
		if !c.Pull {
			return plan.Plan{}, fmt.Errorf("component %q is not pulled", c.Name)
		}
		_, dst := componentPaths(env, c)
		if _, err := env.FS.Lstat(dst); errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, c.Name)
		}
	}
	if len(missing) > 0 {
		return plan.Plan{}, ErrMissingDeployed{Names: missing}
	}
	var steps []plan.Step
	for _, c := range comps {
		repoPath, deployed := componentPaths(env, c)
		info, err := env.FS.Lstat(deployed)
		if err != nil {
			return plan.Plan{}, err
		}
		renderFn, unrenderFn := render.ForComponent(env, c.Target)
		var rows []plan.Change
		switch {
		case info.Mode()&fs.ModeSymlink != 0 && c.Kind.IsDir():
			return plan.Plan{}, fmt.Errorf("refusing to pull through symlink %s", deployed)
		case info.Mode()&fs.ModeSymlink != 0:
			// Diff follows a root source link for install; in pull the deployed side is the source and a link is never copied.
			rows = []plan.Change{{Effect: plan.EffectCreate, Path: filepath.Base(deployed), Note: drift.NoteLink}}
		default:
			if rows, err = drift.DiffWith(env.FS, deployed, repoPath, drift.PullFilter, drift.BlockedFail, unrenderFn); err != nil {
				return plan.Plan{}, err
			}
			for i, row := range rows {
				if row.Effect == plan.EffectOverwrite && staleRendered(env.FS, repoPath, deployed, row.Path, renderFn, unrenderFn) {
					rows[i] = plan.Change{Effect: plan.EffectSkip, Path: row.Path, Note: notPulledStaleNote}
				}
			}
		}
		if step, ok := pullCopy(deployed, repoPath, rows, unrenderFn); ok {
			steps = append(steps, step)
		}
	}
	return plan.New("Pull", store, steps), nil
}

// PullChanges lists the pull components that would copy at least one file into the repo.
// A component whose plan fails other than by a missing deployed path stays listed, so its error shows when planned.
func PullChanges(ctx context.Context, env check.Env, t target.Target) []string {
	var names []string
	for _, c := range t.Manifest.Components {
		if !c.Pull {
			continue
		}
		p, err := PlanPull(ctx, env, t, []string{c.Name})
		if errors.As(err, &ErrMissingDeployed{}) {
			continue
		}
		if err != nil || slices.ContainsFunc(p.Steps, copiesFiles) {
			names = append(names, c.Name)
		}
	}
	return names
}

func copiesFiles(s plan.Step) bool {
	cp, ok := s.(plan.Copy)
	return ok && len(cp.Files) > 0
}

// pullCopy turns Overwrite rows into one Copy; Create rows (deployed-only) and deployed symlinks
// become skip rows that are never copied, so a link cannot land in the repo.
func pullCopy(deployed, repoPath string, rows []plan.Change, transform plan.Transform) (plan.Copy, bool) {
	cp := plan.Copy{Src: deployed, Dst: repoPath, Transform: transform}
	for _, row := range rows {
		switch {
		case row.Note == drift.NoteLink:
			row = plan.Change{Effect: plan.EffectSkip, Path: row.Path, Note: notPulledSymlinkNote}
		case row.Effect == plan.EffectCreate:
			row = plan.Change{Effect: plan.EffectSkip, Path: row.Path, Note: notPulledNote}
		case row.Effect == plan.EffectSkip:
		default:
			cp.Files = append(cp.Files, row.Path)
		}
		cp.Rows = append(cp.Rows, row)
	}
	return cp, len(cp.Rows) > 0
}
