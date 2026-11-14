package drift

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const notExecutableReason = "target is not executable"

// diffStatus maps Diff to a Status: any row or path conflict is drift, named by its first entry.
func diffStatus(env check.Env, c manifest.Component, src, dst string) Status {
	rows, err := Diff(env.FS, src, dst, InstallFilter, BlockedFail)
	var conflict ErrPathConflict
	switch {
	case errors.As(err, &conflict):
		return Status{Name: c.Name, State: StateDrift, Reason: conflictReason(dst, conflict)}
	case err != nil:
		return Status{Name: c.Name, State: StateUnknown, Reason: err.Error()}
	case len(rows) > 0:
		return Status{Name: c.Name, State: StateDrift, Reason: rowReason(rows[0])}
	}
	return Status{Name: c.Name, State: StateInstalled}
}

const notRegularReason = "target is not a regular file"

func rowReason(row plan.Change) string {
	var reason string
	switch {
	case row.Effect == plan.EffectCreate:
		reason = "missing in target"
	case row.Note == NoteLink:
		reason = "source is a symlink"
	case row.Note == NoteReplaces+describeSymlink:
		reason = "target is a symlink"
	case strings.HasPrefix(row.Note, NoteReplaces):
		reason = notRegularReason
	default:
		reason = "content differs"
	}
	if row.Path == rootFile {
		return reason
	}
	return row.Path + ": " + reason
}

func conflictReason(dst string, conflict ErrPathConflict) string {
	name, err := filepath.Rel(dst, conflict.Path)
	if err != nil || name == "." {
		name = "target"
	}
	switch {
	case conflict.Found != describeDirectory:
		return name + " is not a directory"
	case name == "target":
		return notRegularReason
	}
	return name + ": " + notRegularReason
}
