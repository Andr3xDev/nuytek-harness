package specter_test

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestResolveRoot(t *testing.T) {
	const home = "/home/dev"
	tests := []struct {
		name        string
		saved       string
		wantPath    string
		wantSource  specter.RootSource
		wantOutside bool
		wantRelRoot bool
	}{
		{name: "saved wins", saved: "/home/dev/work/specs", wantPath: "/home/dev/work/specs", wantSource: specter.RootSourceSaved},
		{name: "default used", wantPath: "/home/dev/dev/specter", wantSource: specter.RootSourceDefault},
		{name: "saved value is cleaned", saved: "/home/dev/work/../specs/", wantPath: "/home/dev/specs", wantSource: specter.RootSourceSaved},
		{name: "root outside home", saved: "/srv/specter", wantPath: "/srv/specter", wantSource: specter.RootSourceSaved, wantOutside: true},
		{name: "sibling sharing the home prefix is outside", saved: "/home/dev2/specter", wantPath: "/home/dev2/specter", wantSource: specter.RootSourceSaved, wantOutside: true},
		{name: "dotdot-prefixed name inside home is not outside", saved: "/home/dev/..specs", wantPath: "/home/dev/..specs", wantSource: specter.RootSourceSaved},
		{name: "root equal to home is inside", saved: "/home/dev", wantPath: "/home/dev", wantSource: specter.RootSourceSaved},
		{name: "relative root is not outside home", saved: "relative/specter", wantPath: "relative/specter", wantSource: specter.RootSourceSaved, wantRelRoot: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := specter.ResolveRoot(tc.saved, home)

			if got.Path != tc.wantPath || got.Source != tc.wantSource || got.OutsideHome != tc.wantOutside {
				t.Fatalf("ResolveRoot = %+v, want path %q source %q outside %v", got, tc.wantPath, tc.wantSource, tc.wantOutside)
			}
			if got.Relative != tc.wantRelRoot {
				t.Errorf("Relative = %v, want %v", got.Relative, tc.wantRelRoot)
			}
		})
	}
}

func TestLoadRegistration_IgnoresTheRootEnvVar(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.LookupEnv = func(key string) (string, bool) { return "/srv/from-env", key == "NUYTEK_SPECTER_ROOT" }

	reg := specter.LoadRegistration(context.Background(), env)

	if want := env.Home + "/dev/specter"; reg.Root.Path != want || reg.Root.Source != specter.RootSourceDefault {
		t.Errorf("Root = %+v, want the default %q", reg.Root, want)
	}
}
