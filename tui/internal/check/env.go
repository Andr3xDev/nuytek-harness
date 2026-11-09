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

type Exec interface {
	LookPath(name string) (string, error)
	// Output returns combined stdout+stderr.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Env is everything a check may touch.
type Env struct {
	FS         FS
	Exec       Exec
	LookupEnv  func(string) (string, bool)
	Home, Repo string
}
