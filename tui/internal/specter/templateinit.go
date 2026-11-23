package specter

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	templateRepo  = "Andr3xDev/specter-template"
	repoName      = "specter"
	GhTimeout     = 2 * time.Minute
	cloneAttempts = 5
	cloneBackoff  = 2 * time.Second
	// ghNotFoundText is what `gh repo view` prints for an absent repo; a bad token or network failure never contains it.
	ghNotFoundText = "Could not resolve to a Repository"
	shellSafeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_@%+=:,./~-"
)

// InitBlocker is nil when the root can be created from the template, else an ErrBlocked naming the first unmet condition.
func InitBlocker(ctx context.Context, env check.Env, root Root) error {
	_, err := templateOwner(ctx, env, root)
	return err
}

// templateOwner runs the template-init checks in order and returns the GitHub login that will own the new repo.
func templateOwner(ctx context.Context, env check.Env, root Root) (string, error) {
	switch {
	case root.OutsideHome:
		return "", ErrBlocked{noteOutsideHome}
	case root.Relative:
		return "", ErrBlocked{"specter root " + root.Path + " is not an absolute path"}
	}
	if _, err := env.FS.Lstat(root.Path); err == nil {
		if info, serr := env.FS.Stat(root.Path); serr != nil || !info.IsDir() {
			return "", ErrBlocked{"specter root " + root.Path + " exists but is not a directory"}
		}
	}
	for _, tool := range []string{gitBinary, openspecBinary, ghBinary} {
		if _, err := env.Exec.LookPath(tool); err != nil {
			return "", ErrBlocked{tool + " not found on PATH"}
		}
	}
	if res, err := env.Exec.Run(ctx, ghBinary, "auth", "status"); err != nil || res.ExitCode != 0 {
		return "", ErrBlocked{"gh not authenticated; run gh auth login"}
	}
	parent := filepath.Dir(root.Path)
	if info, err := env.FS.Stat(parent); err != nil || !info.IsDir() {
		return "", ErrBlocked{"parent directory " + parent + " does not exist or is not a directory"}
	}

	res, err := env.Exec.Run(ctx, ghBinary, "api", "user", "--jq", ".login")
	owner := strings.TrimSpace(string(res.Stdout))
	if err != nil || res.ExitCode != 0 || owner == "" {
		return "", ErrBlocked{"cannot read the GitHub user: " + ghFailure(res, err)}
	}
	repo := owner + "/" + repoName
	res, err = env.Exec.Run(ctx, ghBinary, "repo", "view", repo)
	switch {
	case err != nil:
		return "", ErrBlocked{"cannot check " + repo + ": " + err.Error()}
	case res.ExitCode == 0:
		return "", ErrBlocked{"repo " + repo + " already exists; clone it with: " + cloneCommand(repo, root.Path)}
	case !bytes.Contains(res.Stderr, []byte(ghNotFoundText)):
		return "", ErrBlocked{"cannot check " + repo + ": " + ghFailure(res, nil)}
	}
	return owner, nil
}

func cloneArgv(repo, dir string) []string { return []string{ghBinary, "repo", "clone", repo, dir} }

func cloneCommand(repo, dir string) string {
	return strings.Join(cloneArgv(repo, shellQuote(dir)), " ")
}

// shellQuote leaves shell-safe words bare and single-quotes anything else.
func shellQuote(s string) string {
	if s != "" && strings.Trim(s, shellSafeChars) == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// planTemplateInit creates the repo from the template and clones it into the root; it never touches an existing root.
func planTemplateInit(ctx context.Context, env check.Env, root Root) (plan.Plan, error) {
	owner, err := templateOwner(ctx, env, root)
	if err != nil {
		return plan.Plan{}, err
	}
	repo := owner + "/" + repoName
	reposPath := filepath.Join(root.Path, reposFile)
	steps := []plan.Step{
		plan.Exec{Argv: []string{ghBinary, "repo", "create", repo, "--template", templateRepo, "--private"}, Timeout: GhTimeout},
		// A clone right after create can fail until the new repo is readable; create is never retried.
		plan.Exec{Argv: cloneArgv(repo, root.Path), Timeout: GhTimeout, Attempts: cloneAttempts, Backoff: cloneBackoff},
		plan.Edit{
			Path: reposPath,
			Edit: func(old []byte, exists bool) ([]byte, error) {
				if !exists {
					// The empty clone already created the root, so only a pull can fill it.
					return nil, fmt.Errorf("%s has no %s (template may still be generating); wait and run: git -C %s pull", root.Path, reposFile, shellQuote(root.Path))
				}
				return old, nil
			},
			Rows: []plan.Change{{Effect: plan.EffectRun, Path: reposPath, Note: "verify " + reposFile + " exists"}},
		},
	}
	return plan.New("Specter fast init", nil, steps), nil
}

// ghFailure is the first stderr line, else the run error, else the exit code.
func ghFailure(res check.ExecResult, err error) string {
	if line, _, _ := strings.Cut(strings.TrimSpace(string(res.Stderr)), "\n"); line != "" {
		return line
	}
	if err != nil {
		return err.Error()
	}
	return "exit code " + strconv.Itoa(res.ExitCode)
}
