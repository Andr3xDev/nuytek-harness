package manifest_test

import (
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

var withSpecter = append([]manifest.Kind{manifest.KindSpecter}, testKinds...)

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
		{"need with relative path", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "./x")}, testKinds, true, `"./x"`},
		{"need with absolute path", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "/usr/bin/x")}, testKinds, true, `"/usr/bin/x"`},
		{"need with whitespace", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "a b")}, testKinds, true, `"a b"`},
		{"need starting with dash", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "-x")}, testKinds, true, `"-x"`},
		{"bare command needs are valid", []manifest.Component{comp("a", manifest.KindFile, "x", ".a", "node", "python3")}, testKinds, false, ""},
		{"dotdot source leading", []manifest.Component{comp("a", manifest.KindFile, "../x", ".a")}, testKinds, true, "../x"},
		{"dotdot source nested", []manifest.Component{comp("a", manifest.KindFile, "a/../../x", ".a")}, testKinds, true, "a/../../x"},
		{"dotdot target", []manifest.Component{comp("a", manifest.KindFile, "x", "../.a")}, testKinds, true, "../.a"},
		{"absolute source", []manifest.Component{comp("a", manifest.KindFile, "/etc/x", ".a")}, testKinds, true, "/etc/x"},
		{"absolute target", []manifest.Component{comp("a", manifest.KindFile, "x", "/etc/a")}, testKinds, true, "/etc/a"},
		{"dots inside segment are valid", []manifest.Component{comp("a", manifest.KindFile, "a..b", ".a..b")}, testKinds, false, ""},
		{"mcp and plugins accepted when allowed", []manifest.Component{comp("mcp", "mcp", "x", ".claude.json"), comp("plugins", "plugins", "y", ".s.json")}, testKinds, false, ""},
		{"mcp rejected when not allowed", []manifest.Component{comp("mcp", "mcp", "x", ".claude.json")}, []manifest.Kind{manifest.KindFile}, true, "mcp"},
		{"nil needs is valid", []manifest.Component{{Name: "a", Kind: manifest.KindFile, Source: "x", Target: ".a"}}, testKinds, false, ""},
		{"pull on mcp rejected", []manifest.Component{{Name: "mcp", Kind: "mcp", Source: "x", Target: ".claude.json", Pull: true}}, testKinds, true, "mcp"},
		{"pull on plugins rejected", []manifest.Component{{Name: "plugins", Kind: "plugins", Source: "x", Target: ".s.json", Pull: true}}, testKinds, true, "plugins"},
		{"pull on dir valid", []manifest.Component{{Name: "agents", Kind: manifest.KindDir, Source: "x", Target: ".a", Pull: true}}, testKinds, false, ""},
		{"non-deploying kind needs no source or target", []manifest.Component{{Name: "sp", Kind: "specter", Needs: []string{"openspec"}}}, withSpecter, false, ""},
		{"non-deploying kind rejects a source", []manifest.Component{{Name: "sp", Kind: "specter", Source: "x"}}, withSpecter, true, "source"},
		{"non-deploying kind rejects a target", []manifest.Component{{Name: "sp", Kind: "specter", Target: ".a"}}, withSpecter, true, "target"},
		{"pull on non-deploying kind rejected", []manifest.Component{{Name: "sp", Kind: "specter", Pull: true}}, withSpecter, true, "pull"},
		{"deploying kind still needs source and target beside a non-deploying one", []manifest.Component{{Name: "sp", Kind: "specter"}, comp("a", manifest.KindFile, "", ".a")}, withSpecter, true, "empty source"},
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
