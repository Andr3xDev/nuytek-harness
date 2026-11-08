package fsx

import (
	"io/fs"
	"os"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
)

var _ app.ReadFS = FS{}

// FS is the os-backed app.ReadFS. It exposes no write method.
type FS struct{}

func (FS) Lstat(path string) (fs.FileInfo, error)     { return os.Lstat(path) }
func (FS) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (FS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
