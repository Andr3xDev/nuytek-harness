package check

import (
	"context"
	"io/fs"
)

// FS has no write method: the read-only guarantee is compile-time.
type FS interface {
	Lstat(path string) (fs.FileInfo, error) // never follows links
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

// ExecResult is a finished command; a non-zero exit is data here, not an error.
type ExecResult struct {
	Stdout, Stderr []byte
	ExitCode       int
}

type Exec interface {
	LookPath(name string) (string, error)
	// Output returns combined stdout+stderr.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// Run returns split streams and the exit code; the error is only for a command that did not start, timed out or was cancelled.
	Run(ctx context.Context, name string, args ...string) (ExecResult, error)
}

// Env is everything a check may touch.
type Env struct {
	FS         FS
	Exec       Exec
	LookupEnv  func(string) (string, bool)
	Home, Repo string
	Cwd        string
}
