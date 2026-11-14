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

	// GIT_TERMINAL_PROMPT=0 makes a credential prompt fail instead of hanging the run; BatchMode does the
	// same for ssh, unless the user already configured how git runs ssh.
	pullArgv := []string{"env", "GIT_TERMINAL_PROMPT=0"}
	if !sshConfigured(ctx, env) {
		pullArgv = append(pullArgv, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	pullArgv = append(pullArgv, "git", "-C", env.Repo, "pull", "--ff-only")
	p := plan.New("Update the TUI", nil, []plan.Step{
		plan.Exec{
			Argv:    pullArgv,
			Timeout: selfUpdatePullTimeout,
		},
		plan.Exec{
			Argv:    []string{"bash", filepath.Join(env.Repo, "scripts", "bootstrap-tui.sh")},
			Timeout: selfUpdateBuildTimeout,
		},
	})
	p.Confirm = "Pull the latest harness and rebuild the TUI?"
	p.Notes = []string{
		fmt.Sprintf("Updates the harness repo at %s%s", env.Repo, currentBranch(ctx, env)),
		"restart the TUI afterwards to run the new build",
	}
	return p, nil
}

// sshConfigured is true when the user already chose the ssh command, which BatchMode injection would override.
func sshConfigured(ctx context.Context, env check.Env) bool {
	for _, name := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		if _, set := env.LookupEnv(name); set {
			return true
		}
	}
	res, err := env.Exec.Run(ctx, "git", "-C", env.Repo, "config", "--get", "core.sshCommand")
	return err == nil && res.ExitCode == 0 && strings.TrimSpace(string(res.Stdout)) != ""
}

// currentBranch returns " (branch <name>)", or "" when the branch cannot be read: it is only informational.
func currentBranch(ctx context.Context, env check.Env) string {
	res, err := env.Exec.Run(ctx, "git", "-C", env.Repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || res.ExitCode != 0 {
		return ""
	}
	if branch := strings.TrimSpace(string(res.Stdout)); branch != "" {
		return fmt.Sprintf(" (branch %s)", branch)
	}
	return ""
}
