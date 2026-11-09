package drift

import (
	"bytes"
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
	return compareStatus(env, c, src, dst, "", c.Kind.NeedsExec())
}

// compareStatus turns the outcome of compareFile into a Status; rel prefixes the drift reason.
func compareStatus(env check.Env, c manifest.Component, src, dst, rel string, wantExec bool) Status {
	reason, err := compareFile(env.FS, src, dst, wantExec)
	switch {
	case err != nil:
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	case reason != "":
		if rel != "" {
			reason = rel + ": " + reason
		}
		return Status{Name: c.Name, State: StateDrift, Reason: reason}
	}
	return Status{Name: c.Name, State: StateInstalled}
}

// compareFile returns why dst differs from src ("" when identical). Symlinks on either side are drift, never followed.
func compareFile(fsys check.FS, src, dst string, wantExec bool) (string, error) {
	srcInfo, err := fsys.Lstat(src)
	if err != nil {
		return "", err
	}
	dstInfo, err := fsys.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "missing in target", nil
	case err != nil:
		return "", err
	case srcInfo.Mode()&fs.ModeSymlink != 0:
		return "source is a symlink", nil
	case dstInfo.Mode()&fs.ModeSymlink != 0:
		return "target is a symlink", nil
	case !dstInfo.Mode().IsRegular():
		return "target is not a regular file", nil
	case wantExec && dstInfo.Mode()&check.OwnerExecBit == 0:
		return "target is not executable", nil
	case srcInfo.Size() != dstInfo.Size():
		return "content differs", nil
	}
	srcData, err := fsys.ReadFile(src)
	if err != nil {
		return "", err
	}
	dstData, err := fsys.ReadFile(dst)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(srcData, dstData) {
		return "content differs", nil
	}
	return "", nil
}
