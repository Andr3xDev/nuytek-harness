package plan

import (
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
)

type Skip struct{ Name, Reason string }

type Plan struct {
	Title   string
	Steps   []Step
	Skipped []Skip
	Notes   []string
	Confirm string
}

// New prepends one Snapshot for the union of Touches; a nil store means snapshot-exempt.
func New(title string, store *backup.Dir, steps []Step) Plan {
	p := Plan{Title: title, Steps: steps}
	if store == nil {
		return p
	}
	var touch Touch
	for _, s := range steps {
		t := s.Touches()
		touch.Paths = append(touch.Paths, t.Paths...)
		touch.Globs = append(touch.Globs, t.Globs...)
	}
	paths := dropCovered(touch.Paths)
	globs := slices.Compact(slices.Sorted(slices.Values(touch.Globs)))
	if len(paths) == 0 && len(globs) == 0 {
		return p
	}
	p.Steps = append([]Step{Snapshot{Store: *store, Paths: paths, Globs: globs}}, steps...)
	return p
}

// dropCovered sorts and dedupes paths, removing any path below another in the set.
func dropCovered(paths []string) []string {
	var kept []string
	for _, path := range slices.Compact(slices.Sorted(slices.Values(paths))) {
		covered := slices.ContainsFunc(kept, func(ancestor string) bool {
			return strings.HasPrefix(path, strings.TrimSuffix(ancestor, "/")+"/")
		})
		if !covered {
			kept = append(kept, path)
		}
	}
	return kept
}

func (p Plan) Rows() []Change {
	var rows []Change
	for _, s := range p.Steps {
		rows = append(rows, s.Changes()...)
	}
	return rows
}

// Empty is true when no row other than EffectSkip exists.
func (p Plan) Empty() bool {
	return !slices.ContainsFunc(p.Rows(), func(c Change) bool { return c.Effect != EffectSkip })
}
