package manifest_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestManifest_AllNeeds(t *testing.T) {
	tests := []struct {
		name string
		m    manifest.Manifest
		want []string
	}{
		{"sorted and de-duplicated", manifest.Manifest{Components: []manifest.Component{
			comp("a", manifest.KindFile, "x", "a", "node", "claude"),
			comp("b", manifest.KindFile, "x", "b", "claude", "jq"),
		}}, []string{"claude", "jq", "node"}},
		{"no components", manifest.Manifest{}, nil},
		{"nil needs only", manifest.Manifest{Components: []manifest.Component{comp("a", manifest.KindFile, "x", "a")}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.AllNeeds()
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Fatalf("got %v, want empty", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManifest_TargetOf(t *testing.T) {
	m := manifest.Manifest{Components: []manifest.Component{comp("settings", manifest.KindFile, "s", ".claude/settings.json")}}
	tests := []struct {
		name   string
		lookup string
		want   string
		wantOK bool
	}{
		{"known", "settings", ".claude/settings.json", true},
		{"unknown", "nope", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := m.TargetOf(tt.lookup)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
