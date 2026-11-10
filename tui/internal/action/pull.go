package action

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const notPulledNote = "not pulled: not in repo"

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
		rows, err := drift.Diff(env.FS, deployed, repoPath, drift.PullFilter, drift.BlockedFail)
		if err != nil {
			return plan.Plan{}, err
		}
		if step, ok := pullCopy(deployed, repoPath, rows); ok {
			steps = append(steps, step)
		}
	}
	return plan.New("Pull", store, steps), nil
}

// pullCopy turns Overwrite rows into one Copy; Create rows (deployed-only) become skip rows that are never copied.
func pullCopy(deployed, repoPath string, rows []plan.Change) (plan.Copy, bool) {
	cp := plan.Copy{Src: deployed, Dst: repoPath}
	for _, row := range rows {
		if row.Effect == plan.EffectCreate {
			row = plan.Change{Effect: plan.EffectSkip, Path: row.Path, Note: notPulledNote}
		} else {
			cp.Files = append(cp.Files, row.Path)
		}
		cp.Rows = append(cp.Rows, row)
	}
	return cp, len(cp.Rows) > 0
}
