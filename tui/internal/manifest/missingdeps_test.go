package manifest_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestMissingDeps(t *testing.T) {
	m := manifest.Manifest{Components: []manifest.Component{
		comp("hooks", manifest.KindDirX, "h", ".h", "python3", "jq"),
		comp("settings", manifest.KindFile, "s", ".s", "jq"),
		comp("agents", manifest.KindDir, "a", ".a", "claude"),
	}}
	tests := []struct {
		name  string
		tools []string
		want  []manifest.MissingDep
	}{
		{"tool missing with dependents", []string{"claude"},
			[]manifest.MissingDep{{Tool: "claude", Dependents: []string{"agents"}}}},
		{"shared tool and deterministic order", []string{"python3", "jq"},
			[]manifest.MissingDep{{Tool: "jq", Dependents: []string{"hooks", "settings"}}, {Tool: "python3", Dependents: []string{"hooks"}}}},
		{"no tools", nil, nil},
		{"undeclared tool", []string{"ghost"},
			[]manifest.MissingDep{{Tool: "ghost"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.MissingDeps(tt.tools)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
