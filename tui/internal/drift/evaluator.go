package drift

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

type Evaluator func(ctx context.Context, env check.Env, c manifest.Component) Status
