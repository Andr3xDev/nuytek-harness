package manifest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidManifestExposesFieldsAndPerComponentNeeds(t *testing.T) {
	path := writeManifest(t, `[[component]]
name   = "agents"
kind   = "dir"
source = "agents"
target = ".claude/agents"
needs  = ["claude"]

[[component]]
name   = "sounds"
kind   = "dir"
source = "claude/sounds"
target = ".claude/sounds"
needs  = []
`)
	m, err := manifest.Load(path, testKinds)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Path != path {
		t.Errorf("Path = %q, want %q", m.Path, path)
	}
	want := []manifest.Component{
		{Name: "agents", Kind: manifest.KindDir, Source: "agents", Target: ".claude/agents", Needs: []string{"claude"}},
		{Name: "sounds", Kind: manifest.KindDir, Source: "claude/sounds", Target: ".claude/sounds"},
	}
	if len(m.Components) != len(want) {
		t.Fatalf("components = %+v, want %+v", m.Components, want)
	}
	for i, w := range want {
		g := m.Components[i]
		if g.Name != w.Name || g.Kind != w.Kind || g.Source != w.Source || g.Target != w.Target || !reflect.DeepEqual(append([]string{}, g.Needs...), append([]string{}, w.Needs...)) {
			t.Errorf("component %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestLoad_AcceptsSpecialKindsPassedByTheTarget(t *testing.T) {
	path := writeManifest(t, `[[component]]
name   = "mcp"
kind   = "mcp"
source = "claude/claude-mcp.json"
target = ".claude.json"
needs  = ["node"]
`)
	m, err := manifest.Load(path, testKinds)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Components) != 1 || m.Components[0].Kind != "mcp" {
		t.Fatalf("components = %+v, want one mcp component", m.Components)
	}
}

func TestLoad_RejectsInvalidManifestWithPathAndLine(t *testing.T) {
	const first = `[[component]]
name   = "agents"
kind   = "dir"
source = "agents"
target = ".claude/agents"
needs  = []

`
	cases := []struct {
		name     string
		second   string
		wantText string
		wantLine int
	}{
		{"unknown field reports its own line", "name = \"two\"\nkind = \"dir\"\nsorce = \"x\"\ntarget = \"t\"\nneeds = []\n", "sorce", 11},
		{"toml syntax error reports its own line", "name = \"two\"\nkind = = \"dir\"\n", "toml", 10},
		{"unknown kind", "name = \"two\"\nkind = \"folder\"\nsource = \"x\"\ntarget = \"t\"\nneeds = []\n", `"folder"`, 8},
		{"duplicate name reports the second header", "name = \"agents\"\nkind = \"dir\"\nsource = \"x\"\ntarget = \"t\"\nneeds = []\n", `"agents"`, 8},
		{"needs missing", "name = \"two\"\nkind = \"dir\"\nsource = \"x\"\ntarget = \"t\"\n", "needs is required (use [] if none)", 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeManifest(t, first+"[[component]]\n"+tc.second)
			_, err := manifest.Load(path, testKinds)
			if err == nil {
				t.Fatal("Load = nil error, want failure")
			}
			wantPos := fmt.Sprintf("%s:%d:", path, tc.wantLine)
			if !strings.Contains(err.Error(), wantPos) {
				t.Errorf("error %q does not contain %q", err, wantPos)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error %q does not contain %q", err, tc.wantText)
			}
		})
	}
}

func TestLoad_MissingFileNamesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.toml")
	_, err := manifest.Load(path, testKinds)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("Load error = %v, want it to name %q", err, path)
	}
}
