package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const unrestorableSentence = "plugin removal cannot be restored from the snapshot"

// namedPart keeps a part next to its component name so confirmations can group by it.
type namedPart struct {
	name string
	plan.Part
}

// partsOf builds the part of each component picked by pick, Late kinds last; errors are joined.
func partsOf(ctx context.Context, env check.Env, t target.Target, comps []manifest.Component, pick func(plan.KindOps) plan.PartFunc) ([]namedPart, error) {
	var parts []namedPart
	var errs []error
	for _, late := range []bool{false, true} {
		for _, c := range comps {
			ops, err := opsFor(t.Spec, c.Kind)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ops.Late != late {
				continue
			}
			build := pick(ops)
			if build == nil {
				errs = append(errs, fmt.Errorf("component %q does not support this action", c.Name))
				continue
			}
			part, err := build(ctx, env, c)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			parts = append(parts, namedPart{name: c.Name, Part: part})
		}
	}
	return parts, errors.Join(errs...)
}

// assemble joins the parts into one plan, so there is one Snapshot for the whole run.
func assemble(title string, store *backup.Dir, parts []namedPart) plan.Plan {
	var steps []plan.Step
	for _, part := range parts {
		steps = append(steps, part.Steps...)
	}
	p := plan.New(title, store, steps)
	for _, part := range parts {
		p.Skipped = append(p.Skipped, part.Skipped...)
		p.Notes = append(p.Notes, part.Notes...)
	}
	return p
}

// confirmDeletions lists every EffectDelete row per component name; the planner's rows are the preview, so both agree.
func confirmDeletions(action string, parts []namedPart) string {
	var b strings.Builder
	unrestorable, deletes := false, false
	for _, part := range parts {
		var rows []plan.Change
		for _, s := range part.Steps {
			for _, row := range s.Changes() {
				if row.Effect == plan.EffectDelete {
					rows = append(rows, row)
					unrestorable = unrestorable || row.Unrestorable
				}
			}
		}
		if len(rows) == 0 {
			continue
		}
		if !deletes {
			fmt.Fprintf(&b, "%s will delete:\n", action)
			deletes = true
		}
		fmt.Fprintf(&b, "%s:\n", part.name)
		for _, row := range rows {
			fmt.Fprintf(&b, "  %s\n", row.Path)
		}
	}
	if !deletes {
		fmt.Fprintf(&b, "%s deletes nothing.\n", action)
	}
	if unrestorable {
		b.WriteString(unrestorableSentence + "\n")
	}
	return b.String()
}
