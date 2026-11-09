package manifest_test

import (
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestValidate_Rejects(t *testing.T) {
	tests := []struct {
		name       string
		components []manifest.Component
		allowed    []manifest.Kind
		wantErr    bool
		wantSubstr string
	}{
		{"valid", []manifest.Component{comp("a", manifest.KindFile, "x/a", ".a", "jq")}, testKinds, false, ""},
		{"unknown kind names the kind", []manifest.Component{comp("a", "folder", "x", ".a")}, testKinds, true, "folder"},
		{"duplicate name names it", []manifest.Component{comp("agents", manifest.KindDir, "x", ".a"), comp("agents", manifest.KindDir, "y", ".b")}, testKinds, true, "agents"},
		{"empty name", []manifest.Component{comp("", manifest.KindFile, "x", ".a")}, testKinds, true, "empty name"},
		{"empty kind", []manifest.Component{comp("a", "", "x", ".a")}, testKinds, true, "empty kind"},
		{"empty source", []manifest.Component{comp("a", manifest.KindFile, "", ".a")}, testKinds, true, "empty source"},
		{"empty target", []manifest.Component{comp("a", manifest.KindFile, "x", "")}, testKinds, true, "empty target"},
		{"dot source", []manifest.Component{comp("a", manifest.KindFile, ".", ".a")}, testKinds, true, `source "."`},
		{"dot-slash source", []manifest.Component{comp("a", manifest.KindFile, "./", ".a")}, testKinds, true, `source "./"`},
		{"dot target", []manifest.Component{comp("a", manifest.KindFile, "x", ".")}, testKinds, true, `target "."`},
		{"empty need", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "jq", "")}, testKinds, true, "empty need"},
		{"dotdot source leading", []manifest.Component{comp("a", manifest.KindFile, "../x", ".a")}, testKinds, true, "../x"},
		{"dotdot source nested", []manifest.Component{comp("a", manifest.KindFile, "a/../../x", ".a")}, testKinds, true, "a/../../x"},
		{"dotdot target", []manifest.Component{comp("a", manifest.KindFile, "x", "../.a")}, testKinds, true, "../.a"},
		{"absolute source", []manifest.Component{comp("a", manifest.KindFile, "/etc/x", ".a")}, testKinds, true, "/etc/x"},
		{"absolute target", []manifest.Component{comp("a", manifest.KindFile, "x", "/etc/a")}, testKinds, true, "/etc/a"},
		{"dots inside segment are valid", []manifest.Component{comp("a", manifest.KindFile, "a..b", ".a..b")}, testKinds, false, ""},
		{"mcp and plugins accepted when allowed", []manifest.Component{comp("mcp", "mcp", "x", ".claude.json"), comp("plugins", "plugins", "y", ".s.json")}, testKinds, false, ""},
		{"mcp rejected when not allowed", []manifest.Component{comp("mcp", "mcp", "x", ".claude.json")}, []manifest.Kind{manifest.KindFile}, true, "mcp"},
		{"nil needs is valid", []manifest.Component{{Name: "a", Kind: manifest.KindFile, Source: "x", Target: ".a"}}, testKinds, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manifest.Validate(manifest.Manifest{Components: tt.components}, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("err %q does not contain %q", err, tt.wantSubstr)
			}
		})
	}
}
