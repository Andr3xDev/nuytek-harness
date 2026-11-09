package dashboard_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func depResult(tool string, status check.Status) check.Result {
	return check.Result{CheckID: "dependencies", Subject: tool, Status: status}
}

func TestMissingDeps(t *testing.T) {
	m := manifest.Manifest{Components: []manifest.Component{
		comp("hooks", manifest.KindDirX, "h", ".h", "python3", "jq"),
		comp("settings", manifest.KindFile, "s", ".s", "jq"),
		comp("agents", manifest.KindDir, "a", ".a", "claude"),
	}}
	tests := []struct {
		name    string
		results []check.Result
		want    []dashboard.MissingDep
	}{
		{"tool missing with dependents", []check.Result{depResult("claude", check.StatusError)},
			[]dashboard.MissingDep{{Tool: "claude", Dependents: []string{"agents"}}}},
		{"shared tool and deterministic order", []check.Result{depResult("python3", check.StatusError), depResult("jq", check.StatusError)},
			[]dashboard.MissingDep{{Tool: "jq", Dependents: []string{"hooks", "settings"}}, {Tool: "python3", Dependents: []string{"hooks"}}}},
		{"all ok", []check.Result{depResult("jq", check.StatusOK), depResult("claude", check.StatusOK)}, nil},
		{"warning and unknown are not missing", []check.Result{depResult("jq", check.StatusWarning), depResult("claude", check.StatusUnknown)}, nil},
		{"error for undeclared tool", []check.Result{depResult("ghost", check.StatusError)},
			[]dashboard.MissingDep{{Tool: "ghost"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dashboard.MissingDeps(m, tt.results)
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Fatalf("got %v, want empty", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i].Tool != tt.want[i].Tool {
					t.Fatalf("got[%d].Tool = %q, want %q", i, got[i].Tool, tt.want[i].Tool)
				}
				if len(tt.want[i].Dependents) == 0 {
					if len(got[i].Dependents) != 0 {
						t.Fatalf("got[%d].Dependents = %v, want empty", i, got[i].Dependents)
					}
				} else if !reflect.DeepEqual(got[i].Dependents, tt.want[i].Dependents) {
					t.Fatalf("got[%d].Dependents = %v, want %v", i, got[i].Dependents, tt.want[i].Dependents)
				}
			}
		})
	}
}
