package dashboard_test

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func resultOf(id string, status check.Status) check.Result {
	return check.Result{CheckID: id, Subject: id, Status: status}
}

func fixedCheck(id string, results ...check.Result) check.Check {
	return check.Check{ID: id, Run: func(context.Context, check.Env) []check.Result { return results }}
}

func targetWith(checks ...check.Check) target.Target {
	return target.Target{Spec: target.Spec{ID: "t"}, Checks: checks}
}

func comp(name string, kind manifest.Kind, source, target string, needs ...string) manifest.Component {
	return manifest.Component{Name: name, Kind: kind, Source: source, Target: target, Needs: needs}
}
