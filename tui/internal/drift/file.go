package drift

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func evaluateFile(_ context.Context, env check.Env, c manifest.Component) Status {
	src, dst := filepath.Join(env.Repo, c.Source), filepath.Join(env.Home, c.Target)
	if _, err := env.FS.Lstat(src); err != nil {
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	}
	if _, err := env.FS.Lstat(dst); errors.Is(err, fs.ErrNotExist) {
		return Status{Name: c.Name, State: StateNotInstalled}
	}
	if s := diffStatus(env, c, src, dst); s.State != StateInstalled {
		return s
	}
	if c.Kind.NeedsExec() && LacksExec(env.FS, dst) {
		return Status{Name: c.Name, State: StateDrift, Reason: notExecutableReason}
	}
	return Status{Name: c.Name, State: StateInstalled}
}
