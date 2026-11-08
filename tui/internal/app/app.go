package app

import (
	"context"
	"io/fs"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

// ReadFS has no write method: the read-only guarantee is compile-time.
type ReadFS interface {
	Lstat(path string) (fs.FileInfo, error) // never follows links
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

type Exec interface {
	LookPath(name string) (string, error)
	// Output returns combined stdout+stderr.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

type StateStore interface {
	// Load returns the state and a warning ("" when none); a missing file is empty state, no warning.
	Load() (domain.State, string)
	Save(domain.State) error
}

// CheckEnv is everything a check may touch.
type CheckEnv struct {
	FS         ReadFS
	Exec       Exec
	LookupEnv  func(string) (string, bool)
	Home, Repo string
}

type Check struct {
	ID  string
	Run func(ctx context.Context, env CheckEnv) []domain.CheckResult
}

type Evaluator func(ctx context.Context, env CheckEnv, c domain.Component) domain.ComponentStatus

// BackupDir is a directory of timestamped backups; Label names it in the dashboard.
type BackupDir struct {
	Path, Label string
}

type TargetSpec struct {
	ID, Name   string
	Evaluators map[domain.Kind]Evaluator              // keys = kinds the manifest may use
	Checks     func(domain.Manifest) ([]Check, error) // nil = planned target, not available
	BackupDirs func(home, repo string) []BackupDir    // nil = none
}

// Target.Err carries manifest and composition errors so one broken target never stops the TUI.
type Target struct {
	Spec     TargetSpec
	Manifest domain.Manifest
	Checks   []Check
	Err      error
}

type BackupInfo struct {
	Time  time.Time
	Label string
}

type Dashboard struct {
	Counts      map[domain.Status]int
	Results     []domain.CheckResult
	Components  []domain.ComponentStatus
	MissingDeps []domain.MissingDep
	LastBackup  *BackupInfo // nil = no backups
	Err         error
}

// Planned returns a spec for a target that is announced but not implemented.
func Planned(id, name string) TargetSpec {
	return TargetSpec{ID: id, Name: name}
}
