package drift

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

const scriptSuffix = ".sh"

// evaluateDir walks the repo source only, so files that exist only in the target never count as drift.
func evaluateDir(_ context.Context, env check.Env, c manifest.Component) Status {
	src, dst := filepath.Join(env.Repo, c.Source), filepath.Join(env.Home, c.Target)
	srcInfo, err := env.FS.Lstat(src)
	if err != nil || !srcInfo.IsDir() {
		return Status{Name: c.Name, State: StateUnknown, Reason: "source is not a readable directory: " + c.Source}
	}
	dstInfo, err := env.FS.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Status{Name: c.Name, State: StateNotInstalled}
	case err != nil:
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	case !dstInfo.IsDir():
		return Status{Name: c.Name, State: StateDrift, Reason: "target is not a directory"}
	}
	return walkDir(env, c, src, dst, "")
}

// walkDir returns the first drift found under src/dst; rel is the slash-separated path from the component root.
func walkDir(env check.Env, c manifest.Component, src, dst, rel string) Status {
	entries, err := env.FS.ReadDir(src)
	if err != nil {
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	}
	for _, e := range entries {
		name := e.Name()
		if InstallFilter.Excludes(name) {
			continue
		}
		entryRel := path.Join(rel, name)
		entrySrc, entryDst := filepath.Join(src, name), filepath.Join(dst, name)
		if e.IsDir() {
			// A symlinked target subdirectory would make the file lookups below follow it.
			if info, err := env.FS.Lstat(entryDst); err == nil && !info.IsDir() {
				return Status{Name: c.Name, State: StateDrift, Reason: entryRel + " is not a directory"}
			}
			if s := walkDir(env, c, entrySrc, entryDst, entryRel); s.State != StateInstalled {
				return s
			}
			continue
		}
		// Only top-level scripts are executed directly by the installer's dirx contract.
		wantExec := c.Kind == manifest.KindDirX && rel == "" && strings.HasSuffix(name, scriptSuffix)
		if s := compareStatus(env, c, entrySrc, entryDst, entryRel, wantExec); s.State != StateInstalled {
			return s
		}
	}
	return Status{Name: c.Name, State: StateInstalled}
}
