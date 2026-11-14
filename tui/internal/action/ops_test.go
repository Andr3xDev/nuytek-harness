package action

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func TestOpsForResolvesEveryEvaluatedKind(t *testing.T) {
	for kind := range drift.Evaluators(nil) {
		if _, err := opsFor(target.Spec{}, kind); err != nil {
			t.Errorf("opsFor(%q) = %v, want built-in ops", kind, err)
		}
	}
}

func TestOpsForBuiltInData(t *testing.T) {
	tests := []struct {
		kind     manifest.Kind
		wantHard bool
	}{
		{manifest.KindDir, true},
		{manifest.KindDirX, true},
		{manifest.KindFile, false},
		{manifest.KindFileX, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			ops, err := opsFor(target.Spec{}, tt.kind)

			if err != nil {
				t.Fatal(err)
			}
			if ops.Install == nil || ops.Remove == nil || (ops.Hard != nil) != tt.wantHard {
				t.Fatalf("Install set %v, Remove set %v, Hard set %v; want Hard set %v",
					ops.Install != nil, ops.Remove != nil, ops.Hard != nil, tt.wantHard)
			}
		})
	}
}

func TestOpsForPrefersSpecialAndRejectsUnknown(t *testing.T) {
	special := plan.KindOps{Late: true}
	spec := target.Spec{Special: map[manifest.Kind]plan.KindOps{"mcp": special}}

	if ops, err := opsFor(spec, "mcp"); err != nil || !ops.Late {
		t.Errorf("opsFor(mcp) = (%+v, %v), want the special ops", ops, err)
	}
	if _, err := opsFor(spec, "unknown"); err == nil {
		t.Error("opsFor(unknown) error = nil, want an error")
	}
}

func TestHardOrInstall(t *testing.T) {
	part := func(note string) plan.PartFunc {
		return func(context.Context, check.Env, manifest.Component) (plan.Part, error) {
			return plan.Part{Notes: []string{note}}, nil
		}
	}
	tests := []struct {
		name string
		ops  plan.KindOps
		want string
	}{
		{"hard unset falls back to install", plan.KindOps{Install: part("install")}, "install"},
		{"hard set wins", plan.KindOps{Install: part("install"), Hard: part("hard")}, "hard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hardOrInstall(tt.ops)(context.Background(), check.Env{}, manifest.Component{})

			if err != nil || len(got.Notes) != 1 || got.Notes[0] != tt.want {
				t.Fatalf("part = (%+v, %v), want note %q", got, err, tt.want)
			}
		})
	}
}
