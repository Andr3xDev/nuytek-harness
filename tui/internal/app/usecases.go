package app

import (
	"context"
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

// DependenciesCheckID tags the check results that ComputeDashboard turns into MissingDeps.
const DependenciesCheckID = "dependencies"

const MaxConcurrentEvaluations = 4

const (
	manifestExt = ".toml"
	targetsDir  = "targets"
)

// RunChecks runs every check concurrently; a panicking check becomes one error result.
func RunChecks(ctx context.Context, env CheckEnv, checks []Check) []domain.CheckResult {
	perCheck := make([][]domain.CheckResult, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					perCheck[i] = []domain.CheckResult{{CheckID: c.ID, Subject: c.ID, Status: domain.StatusError, Message: fmt.Sprintf("check panicked: %v", r)}}
				}
			}()
			perCheck[i] = c.Run(ctx, env)
		}()
	}
	wg.Wait()
	return slices.Concat(perCheck...)
}

func ComputeDashboard(ctx context.Context, env CheckEnv, t Target) Dashboard {
	d := Dashboard{Counts: map[domain.Status]int{}, Err: t.Err}
	if t.Err != nil {
		return d
	}
	d.Results = RunChecks(ctx, env, t.Checks)
	var deps []domain.CheckResult
	for _, r := range d.Results {
		d.Counts[r.Status]++
		if r.CheckID == DependenciesCheckID {
			deps = append(deps, r)
		}
	}
	d.MissingDeps = domain.MissingDeps(t.Manifest, deps)
	d.Components = evaluateComponents(ctx, env, t)
	if t.Spec.BackupDirs != nil {
		d.LastBackup = LastBackup(env.FS, t.Spec.BackupDirs(env.Home, env.Repo))
	}
	return d
}

func evaluateComponents(ctx context.Context, env CheckEnv, t Target) []domain.ComponentStatus {
	statuses := make([]domain.ComponentStatus, len(t.Manifest.Components))
	slots := make(chan struct{}, MaxConcurrentEvaluations)
	var wg sync.WaitGroup
	for i, c := range t.Manifest.Components {
		evaluate := t.Spec.Evaluators[c.Kind]
		if evaluate == nil {
			statuses[i] = domain.ComponentStatus{Name: c.Name, State: domain.StateUnknown}
			continue
		}
		slots <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			defer func() {
				if r := recover(); r != nil {
					statuses[i] = domain.ComponentStatus{Name: c.Name, State: domain.StateUnknown, Reason: fmt.Sprintf("evaluator panicked: %v", r)}
				}
			}()
			statuses[i] = evaluate(ctx, env, c)
		}()
	}
	wg.Wait()
	return statuses
}

// Issues returns the non-ok results, errors first.
func (d Dashboard) Issues() []domain.CheckResult {
	var errs, others []domain.CheckResult
	for _, r := range d.Results {
		switch r.Status {
		case domain.StatusOK:
		case domain.StatusError:
			errs = append(errs, r)
		default:
			others = append(others, r)
		}
	}
	return append(errs, others...)
}

// LastBackup returns the newest backup across dirs, or nil when there is none.
func LastBackup(fs ReadFS, dirs []BackupDir) *BackupInfo {
	var newest *BackupInfo
	for _, dir := range dirs {
		entries, err := fs.ReadDir(dir.Path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			when, ok := domain.ParseBackupStamp(e.Name())
			if !ok {
				info, err := e.Info()
				if err != nil {
					continue
				}
				when = info.ModTime()
			}
			if newest == nil || when.After(newest.Time) {
				newest = &BackupInfo{Time: when, Label: dir.Label}
			}
		}
	}
	return newest
}

// Resolution sources, the contract with the UI and main.
const (
	SourceFlag  = "flag"
	SourceEnv   = "env"
	SourceState = "state"
	SourceCwd   = "cwd"
)

type Resolution struct {
	Path, Source, Warning string
}

// isRepo reports whether dir holds targets/<marker>.
func isRepo(fs ReadFS, dir, marker string) bool {
	_, err := fs.Lstat(filepath.Join(dir, targetsDir, marker))
	return err == nil
}

// resolveExplicit validates a flag or env repo path and makes it absolute so it is never saved relative.
func resolveExplicit(label, path, source, marker string, fs ReadFS) (Resolution, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Resolution{}, fmt.Errorf("%s %s: %w", label, path, err)
	}
	if !isRepo(fs, abs, marker) {
		return Resolution{}, fmt.Errorf("%s %s is not a harness repo (missing %s/%s)", label, abs, targetsDir, marker)
	}
	return Resolution{Path: abs, Source: source}, nil
}

// ResolveRepo picks the repo by precedence flag > env > state > walking up from cwd.
// marker is the file name inside targets/ that identifies a harness repo.
// Explicit flag/env without a repo marker is an error; a stale state path only warns.
func ResolveRepo(flag, env string, st domain.State, cwd, marker string, fs ReadFS) (Resolution, error) {
	if flag != "" {
		return resolveExplicit("--repo", flag, SourceFlag, marker, fs)
	}
	if env != "" {
		return resolveExplicit("NUYTEK_HARNESS_REPO", env, SourceEnv, marker, fs)
	}
	var warning string
	if st.RepoPath != "" {
		if isRepo(fs, st.RepoPath, marker) {
			return Resolution{Path: st.RepoPath, Source: SourceState}, nil
		}
		warning = fmt.Sprintf("saved repo_path %s is no longer a harness repo; ignoring it", st.RepoPath)
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if isRepo(fs, dir, marker) {
			return Resolution{Path: dir, Source: SourceCwd, Warning: warning}, nil
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return Resolution{}, fmt.Errorf("no harness repo found: pass --repo, set NUYTEK_HARNESS_REPO, set repo_path in the state file, or run from inside the repo (current directory %s)", cwd)
}

type LoadManifest func(path string, kinds []domain.Kind) (domain.Manifest, error)

// LoadTargets returns the registered specs first, then manifests found in targets/ that no spec claims.
func LoadTargets(repo string, specs []TargetSpec, load LoadManifest, fs ReadFS) []Target {
	targets := make([]Target, 0, len(specs))
	registered := map[string]bool{}
	for _, spec := range specs {
		registered[spec.ID] = true
		targets = append(targets, loadTarget(repo, spec, load, fs))
	}
	entries, err := fs.ReadDir(filepath.Join(repo, targetsDir))
	if err != nil && !errors.Is(err, iofs.ErrNotExist) {
		// An unreadable targets dir would otherwise hide every manifest; show it as a failed target.
		targets = append(targets, Target{Spec: Planned(targetsDir, targetsDir), Err: fmt.Errorf("reading %s: %w", filepath.Join(repo, targetsDir), err)})
	}
	for _, e := range entries { // ReadDir is sorted by name
		id, isManifest := strings.CutSuffix(e.Name(), manifestExt)
		if isManifest && !registered[id] {
			targets = append(targets, Target{Spec: Planned(id, id)})
		}
	}
	return targets
}

func loadTarget(repo string, spec TargetSpec, load LoadManifest, fs ReadFS) Target {
	t := Target{Spec: spec}
	if spec.Checks == nil {
		return t
	}
	path := filepath.Join(repo, targetsDir, spec.ID+manifestExt)
	if _, err := fs.Lstat(path); err != nil {
		t.Err = fmt.Errorf("manifest %s: %w", path, err)
		return t
	}
	kinds := make([]domain.Kind, 0, len(spec.Evaluators))
	for k := range spec.Evaluators {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	m, err := load(path, kinds)
	if err != nil {
		t.Err = fmt.Errorf("manifest %s: %w", path, err)
		return t
	}
	t.Manifest = m
	if t.Checks, err = spec.Checks(m); err != nil {
		t.Err = fmt.Errorf("composing checks for %s: %w", spec.ID, err)
	}
	return t
}
