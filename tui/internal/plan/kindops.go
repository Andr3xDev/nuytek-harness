package plan

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

type Part struct {
	Steps   []Step
	Skipped []Skip
	Notes   []string
}

type PartFunc func(ctx context.Context, env check.Env, c manifest.Component) (Part, error)

type KindOps struct {
	Install        PartFunc
	Remove         PartFunc // nil = not removable
	Hard           PartFunc // nil = Hard install runs Install
	Late           bool     // planned after every other component
	ExplicitUpdate bool     // never preselected, never in "update all"
}
