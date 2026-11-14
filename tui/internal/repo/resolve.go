package repo

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// FS is the only filesystem access Resolve needs.
type FS interface {
	Lstat(path string) (fs.FileInfo, error)
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

func isRepo(fsys FS, dir, marker string) bool {
	_, err := fsys.Lstat(filepath.Join(dir, marker))
	return err == nil
}

// FindUp returns the nearest directory at or above from that contains marker.
func FindUp(fsys FS, from, marker string) (string, bool) {
	for dir := from; ; dir = filepath.Dir(dir) {
		if isRepo(fsys, dir, marker) {
			return dir, true
		}
		if filepath.Dir(dir) == dir {
			return "", false
		}
	}
}

// resolveExplicit validates a flag or env repo path and makes it absolute so it is never saved relative.
func resolveExplicit(label, path, source, marker string, fsys FS) (Resolution, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Resolution{}, fmt.Errorf("%s %s: %w", label, path, err)
	}
	if !isRepo(fsys, abs, marker) {
		return Resolution{}, fmt.Errorf("%s %s is not a harness repo (missing %s)", label, abs, marker)
	}
	return Resolution{Path: abs, Source: source}, nil
}

// Resolve picks the repo by precedence flag > env > statePath > walking up from cwd.
// marker is the repo-relative file path that identifies a harness repo.
// Explicit flag/env without a repo marker is an error; a stale statePath only warns.
func Resolve(flag, env, statePath, cwd, marker string, fsys FS) (Resolution, error) {
	if flag != "" {
		return resolveExplicit("--repo", flag, SourceFlag, marker, fsys)
	}
	if env != "" {
		return resolveExplicit("NUYTEK_HARNESS_REPO", env, SourceEnv, marker, fsys)
	}
	var warning string
	if statePath != "" {
		if isRepo(fsys, statePath, marker) {
			return Resolution{Path: statePath, Source: SourceState}, nil
		}
		warning = fmt.Sprintf("saved repo_path %s is no longer a harness repo; ignoring it", statePath)
	}
	if dir, ok := FindUp(fsys, cwd, marker); ok {
		return Resolution{Path: dir, Source: SourceCwd, Warning: warning}, nil
	}
	return Resolution{}, fmt.Errorf("no harness repo found: pass --repo, set NUYTEK_HARNESS_REPO, set repo_path in the state file, or run from inside the repo (current directory %s)", cwd)
}
