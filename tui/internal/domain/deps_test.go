package domain

import (
	"reflect"
	"testing"
)

func depResult(tool string, status Status) CheckResult {
	return CheckResult{CheckID: "dependencies", Subject: tool, Status: status}
}

func TestMissingDeps(t *testing.T) {
	m := Manifest{Components: []Component{
		comp("hooks", KindDirX, "h", ".h", "python3", "jq"),
		comp("settings", KindFile, "s", ".s", "jq"),
		comp("agents", KindDir, "a", ".a", "claude"),
	}}
	tests := []struct {
		name    string
		results []CheckResult
		want    []MissingDep
	}{
		{"tool missing with dependents", []CheckResult{depResult("claude", StatusError)},
			[]MissingDep{{Tool: "claude", Dependents: []string{"agents"}}}},
		{"shared tool and deterministic order", []CheckResult{depResult("python3", StatusError), depResult("jq", StatusError)},
			[]MissingDep{{Tool: "jq", Dependents: []string{"hooks", "settings"}}, {Tool: "python3", Dependents: []string{"hooks"}}}},
		{"all ok", []CheckResult{depResult("jq", StatusOK), depResult("claude", StatusOK)}, nil},
		{"warning and unknown are not missing", []CheckResult{depResult("jq", StatusWarning), depResult("claude", StatusUnknown)}, nil},
		{"error for undeclared tool", []CheckResult{depResult("ghost", StatusError)},
			[]MissingDep{{Tool: "ghost"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MissingDeps(m, tt.results)
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
