package specter_test

import (
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestResolveRoot(t *testing.T) {
	const home = "/home/dev"
	tests := []struct {
		name        string
		env         map[string]string
		wantPath    string
		wantSource  specter.RootSource
		wantOutside bool
	}{
		{"env var wins", map[string]string{rootEnvVar: "/home/dev/work/specs"}, "/home/dev/work/specs", specter.RootSourceEnv, false},
		{"default used", nil, "/home/dev/dev/specter", specter.RootSourceDefault, false},
		{"empty env var is unset", map[string]string{rootEnvVar: ""}, "/home/dev/dev/specter", specter.RootSourceDefault, false},
		{"env value is cleaned", map[string]string{rootEnvVar: "/home/dev/work/../specs/"}, "/home/dev/specs", specter.RootSourceEnv, false},
		{"root outside home", map[string]string{rootEnvVar: "/srv/specter"}, "/srv/specter", specter.RootSourceEnv, true},
		{"sibling sharing the home prefix is outside", map[string]string{rootEnvVar: "/home/dev2/specter"}, "/home/dev2/specter", specter.RootSourceEnv, true},
		{"dotdot-prefixed name inside home is not outside", map[string]string{rootEnvVar: "/home/dev/..specs"}, "/home/dev/..specs", specter.RootSourceEnv, false},
		{"root equal to home is inside", map[string]string{rootEnvVar: "/home/dev"}, "/home/dev", specter.RootSourceEnv, false},
		{"relative root is not outside home", map[string]string{rootEnvVar: "relative/specter"}, "relative/specter", specter.RootSourceEnv, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) { v, ok := tc.env[key]; return v, ok }

			got := specter.ResolveRoot(lookup, home)

			if got.Path != tc.wantPath || got.Source != tc.wantSource || got.OutsideHome != tc.wantOutside {
				t.Fatalf("ResolveRoot = %+v, want path %q source %q outside %v", got, tc.wantPath, tc.wantSource, tc.wantOutside)
			}
			if wantRelative := tc.name == "relative root is not outside home"; got.Relative != wantRelative {
				t.Errorf("Relative = %v, want %v", got.Relative, wantRelative)
			}
		})
	}
}
