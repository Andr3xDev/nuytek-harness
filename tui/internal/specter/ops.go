package specter

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// Ops only installs: Hard is nil so Hard install runs Install (it never unregisters), Remove is nil so the harness never removes stores.
var Ops = plan.KindOps{Install: installRegistration}

// installRegistration turns a missing or unsupported root into a Skip, so install-all keeps the other components.
func installRegistration(ctx context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	reg := LoadRegistration(ctx, env)
	if reg.Root.OutsideHome || !reg.RootExists {
		return plan.Part{Skipped: []plan.Skip{{Name: c.Name, Reason: registerBlocker(reg)}}}, nil
	}
	return registerPart(reg), nil
}

// Evaluate never reports drift: an unregistered store is not_installed, and drift detail lives in Drift(f).
func Evaluate(ctx context.Context, env check.Env, c manifest.Component) drift.Status {
	status := func(state drift.State, reason string) drift.Status {
		return drift.Status{Name: c.Name, State: state, Reason: reason}
	}
	reg := LoadRegistration(ctx, env)
	switch {
	case reg.Root.OutsideHome:
		return status(drift.StateSkipped, noteOutsideHome)
	case !reg.Openspec.Found:
		return status(drift.StateSkipped, errOpenspecMissing.Error())
	case !reg.RootExists && reg.FoldersErr == nil:
		return status(drift.StateSkipped, "specter root not found")
	case reg.RegisteredErr != nil:
		return status(drift.StateUnknown, reg.RegisteredErr.Error())
	case reg.FoldersErr != nil:
		return status(drift.StateUnknown, reg.FoldersErr.Error())
	case len(reg.Folders) == 0:
		return status(drift.StateSkipped, "no stores found")
	}
	for _, folder := range reg.Folders {
		if root, registered := registeredAt(reg, folder.ID); folder.ID != "" && (!registered || !sameDir(root, folder.Dir)) {
			return status(drift.StateNotInstalled, "")
		}
	}
	return status(drift.StateInstalled, "")
}
