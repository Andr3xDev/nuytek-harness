package drift_test

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestEvaluators_MergesExtraAndKeepsDataKinds(t *testing.T) {
	called := false
	extra := map[manifest.Kind]drift.Evaluator{
		"mcp": func(context.Context, check.Env, manifest.Component) drift.Status {
			called = true
			return drift.Status{State: drift.StateDrift}
		},
	}

	got := drift.Evaluators(extra)

	for _, kind := range []manifest.Kind{manifest.KindDir, manifest.KindDirX, manifest.KindFile, manifest.KindFileX, "mcp"} {
		if got[kind] == nil {
			t.Fatalf("no evaluator for kind %q", kind)
		}
	}
	got["mcp"](context.Background(), check.Env{}, manifest.Component{})
	if !called {
		t.Error("extra evaluator was replaced")
	}
}
