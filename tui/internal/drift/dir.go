package drift

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

// evaluateDir diffs the repo source only, so files that exist only in the target never count as drift.
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
	if s := diffStatus(env, c, src, dst); s.State != StateInstalled || c.Kind != manifest.KindDirX {
		return s
	}
	// Only top-level scripts are executed directly by the installer's dirx contract.
	scripts, err := Scripts(env.FS, src, InstallFilter)
	if err != nil {
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	}
	for _, name := range scripts {
		if LacksExec(env.FS, filepath.Join(dst, name)) {
			return Status{Name: c.Name, State: StateDrift, Reason: name + ": " + notExecutableReason}
		}
	}
	return Status{Name: c.Name, State: StateInstalled}
}
