package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

func TestSaveRepoPath_KeepsSavedConfigAndSetsRepo(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.toml"))
	saved := state.State{
		RepoPath:    "/old",
		SpecterRoot: "/data/specs",
		Targets:     map[string]state.TargetState{"claude": {BackupsDir: "/data/bk"}},
	}
	if err := store.Save(saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := saveRepoPath(store, saved, "/a"); err != nil {
		t.Fatalf("saveRepoPath: %v", err)
	}
	want := saved
	want.RepoPath = "/a"
	got, warning := store.Load()
	if !reflect.DeepEqual(got, want) || warning != "" {
		t.Fatalf("Load = %+v, %q; want %+v", got, warning, want)
	}
}
