package action

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	selfUpdatePullTimeout  = 2 * time.Minute
	selfUpdateBuildTimeout = 5 * time.Minute
)

// PlanSelfUpdate pulls the harness repo and rebuilds the TUI; refuses on a dirty tree.
func PlanSelfUpdate(ctx context.Context, env check.Env) (plan.Plan, error) {
	res, err := env.Exec.Run(ctx, "git", "-C", env.Repo, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return plan.Plan{}, fmt.Errorf("read harness repo status: %w", err)
	}
	if res.ExitCode != 0 {
		return plan.Plan{}, fmt.Errorf("read harness repo status: %s", strings.TrimSpace(string(res.Stderr)))
	}
	if len(strings.TrimSpace(string(res.Stdout))) > 0 {
		return plan.Plan{}, fmt.Errorf("harness repo has uncommitted changes")
	}

	// GIT_TERMINAL_PROMPT=0 makes a credential prompt fail instead of hanging the run.
	p := plan.New("Update the TUI", nil, []plan.Step{
		plan.Exec{
			Argv:    []string{"env", "GIT_TERMINAL_PROMPT=0", "git", "-C", env.Repo, "pull", "--ff-only"},
			Timeout: selfUpdatePullTimeout,
		},
		plan.Exec{
			Argv:    []string{"bash", filepath.Join(env.Repo, "scripts", "bootstrap-tui.sh")},
			Timeout: selfUpdateBuildTimeout,
		},
	})
	p.Confirm = "Pull the latest harness and rebuild the TUI?"
	p.Notes = []string{
		fmt.Sprintf("Updates the harness repo at %s", env.Repo),
		"restart the TUI afterwards to run the new build",
	}
	return p, nil
}
