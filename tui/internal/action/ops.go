package action

import (
	"fmt"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// opsFor returns the special ops of the kind, else the built-in data ops.
func opsFor(spec target.Spec, kind manifest.Kind) (plan.KindOps, error) {
	if ops, ok := spec.Special[kind]; ok {
		return ops, nil
	}
	if !kind.IsData() {
		return plan.KindOps{}, fmt.Errorf("no ops for component kind %q", kind)
	}
	// file and filex already overwrite whole, so Hard install runs their Install part.
	ops := plan.KindOps{Install: installPart, Remove: removePart}
	if kind.IsDir() {
		ops.Hard = hardDirPart
	}
	return ops, nil
}

// hardOrInstall returns ops.Hard when set, else ops.Install.
func hardOrInstall(ops plan.KindOps) plan.PartFunc {
	if ops.Hard != nil {
		return ops.Hard
	}
	return ops.Install
}
