package domain

import (
	"reflect"
	"strings"
	"testing"
)

var testKinds = []Kind{KindDir, KindDirX, KindFile, KindFileX, "mcp", "plugins"}

func comp(name string, kind Kind, source, target string, needs ...string) Component {
	return Component{Name: name, Kind: kind, Source: source, Target: target, Needs: needs}
}

func TestValidateManifest_Rejects(t *testing.T) {
	tests := []struct {
		name       string
		components []Component
		allowed    []Kind
		wantErr    bool
		wantSubstr string
	}{
		{"valid", []Component{comp("a", KindFile, "x/a", ".a", "jq")}, testKinds, false, ""},
		{"unknown kind names the kind", []Component{comp("a", "folder", "x", ".a")}, testKinds, true, "folder"},
		{"duplicate name names it", []Component{comp("agents", KindDir, "x", ".a"), comp("agents", KindDir, "y", ".b")}, testKinds, true, "agents"},
		{"empty name", []Component{comp("", KindFile, "x", ".a")}, testKinds, true, "empty name"},
		{"empty kind", []Component{comp("a", "", "x", ".a")}, testKinds, true, "empty kind"},
		{"empty source", []Component{comp("a", KindFile, "", ".a")}, testKinds, true, "empty source"},
		{"empty target", []Component{comp("a", KindFile, "x", "")}, testKinds, true, "empty target"},
		{"dot source", []Component{comp("a", KindFile, ".", ".a")}, testKinds, true, `source "."`},
		{"dot-slash source", []Component{comp("a", KindFile, "./", ".a")}, testKinds, true, `source "./"`},
		{"dot target", []Component{comp("a", KindFile, "x", ".")}, testKinds, true, `target "."`},
		{"empty need", []Component{comp("a", KindFile, "x", ".a", "jq", "")}, testKinds, true, "empty need"},
		{"dotdot source leading", []Component{comp("a", KindFile, "../x", ".a")}, testKinds, true, "../x"},
		{"dotdot source nested", []Component{comp("a", KindFile, "a/../../x", ".a")}, testKinds, true, "a/../../x"},
		{"dotdot target", []Component{comp("a", KindFile, "x", "../.a")}, testKinds, true, "../.a"},
		{"absolute source", []Component{comp("a", KindFile, "/etc/x", ".a")}, testKinds, true, "/etc/x"},
		{"absolute target", []Component{comp("a", KindFile, "x", "/etc/a")}, testKinds, true, "/etc/a"},
		{"dots inside segment are valid", []Component{comp("a", KindFile, "a..b", ".a..b")}, testKinds, false, ""},
		{"mcp and plugins accepted when allowed", []Component{comp("mcp", "mcp", "x", ".claude.json"), comp("plugins", "plugins", "y", ".s.json")}, testKinds, false, ""},
		{"mcp rejected when not allowed", []Component{comp("mcp", "mcp", "x", ".claude.json")}, []Kind{KindFile}, true, "mcp"},
		{"nil needs is valid", []Component{{Name: "a", Kind: KindFile, Source: "x", Target: ".a"}}, testKinds, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest(Manifest{Components: tt.components}, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("err %q does not contain %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestManifest_AllNeeds(t *testing.T) {
	tests := []struct {
		name string
		m    Manifest
		want []string
	}{
		{"sorted and de-duplicated", Manifest{Components: []Component{
			comp("a", KindFile, "x", "a", "node", "claude"),
			comp("b", KindFile, "x", "b", "claude", "jq"),
		}}, []string{"claude", "jq", "node"}},
		{"no components", Manifest{}, nil},
		{"nil needs only", Manifest{Components: []Component{comp("a", KindFile, "x", "a")}}, nil},
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
	m := Manifest{Components: []Component{comp("settings", KindFile, "s", ".claude/settings.json")}}
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

func TestKind_IsDirNeedsExec(t *testing.T) {
	tests := []struct {
		kind     Kind
		wantDir  bool
		wantExec bool
	}{
		{KindDir, true, false},
		{KindDirX, true, true},
		{KindFile, false, false},
		{KindFileX, false, true},
		{"mcp", false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsDir(); got != tt.wantDir {
				t.Errorf("IsDir = %v, want %v", got, tt.wantDir)
			}
			if got := tt.kind.NeedsExec(); got != tt.wantExec {
				t.Errorf("NeedsExec = %v, want %v", got, tt.wantExec)
			}
		})
	}
}
