package specter_test

import (
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestFastInitBlocker(t *testing.T) {
	ready := specter.Facts{
		Registration: specter.Registration{Openspec: specter.ToolInfo{Found: true}},
		Git:          specter.ToolInfo{Found: true},
		Gh:           specter.GhInfo{Found: true, Authenticated: true},
	}
	tests := []struct {
		name string
		edit func(*specter.Facts)
		want string // "" = no blocker
	}{
		{"all present", func(*specter.Facts) {}, ""},
		{"outside HOME", func(f *specter.Facts) { f.Root.OutsideHome = true }, "outside HOME"},
		{"git missing", func(f *specter.Facts) { f.Git.Found = false }, "git"},
		{"openspec missing", func(f *specter.Facts) { f.Openspec.Found = false }, "openspec"},
		{"gh missing", func(f *specter.Facts) { f.Gh = specter.GhInfo{} }, "gh not found"},
		{"gh unauthenticated", func(f *specter.Facts) { f.Gh.Authenticated = false }, "not authenticated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ready
			tt.edit(&f)

			got := specter.FastInitBlocker(f)

			if (tt.want == "") != (got == "") || !strings.Contains(got, tt.want) {
				t.Errorf("FastInitBlocker = %q, want %q mentioned", got, tt.want)
			}
		})
	}
}
