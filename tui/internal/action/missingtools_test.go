package action_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestMissingTools(t *testing.T) {
	m := manifest.Manifest{Components: []manifest.Component{
		{Name: "a", Needs: []string{"x"}},
		{Name: "b", Needs: []string{"x", "y"}},
		{Name: "c"},
	}}
	tests := []struct {
		name    string
		missing []string
		want    []manifest.MissingDep
	}{
		{"all present", nil, nil},
		{"one missing", []string{"y"}, []manifest.MissingDep{{Tool: "y", Dependents: []string{"b"}}}},
		{"two missing", []string{"y", "x"}, []manifest.MissingDep{
			{Tool: "x", Dependents: []string{"a", "b"}}, {Tool: "y", Dependents: []string{"b"}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := action.MissingTools(fakeExec{missing: tt.missing}, m)

			if len(got) != len(tt.want) || (len(got) > 0 && !reflect.DeepEqual(got, tt.want)) {
				t.Fatalf("MissingTools = %+v, want %+v", got, tt.want)
			}
		})
	}
}
