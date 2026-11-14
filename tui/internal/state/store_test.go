package state_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/testutil"
)

// Both stores must satisfy the interface the shell consumes.
type store interface {
	Load() (state.State, string)
	Save(state.State) error
}

var (
	_ store = (*state.Store)(nil)
	_ store = state.DiscardStore{}
)

func TestStore_MissingFileLoadsEmptyWithoutWarning(t *testing.T) {
	got, warning := state.NewStore(statePath(t)).Load()
	if got != (state.State{}) || warning != "" {
		t.Fatalf("Load = %+v, %q; want empty state and no warning", got, warning)
	}
}

func TestStore_SavedFileIsTOMLWithRepoPathOnly(t *testing.T) {
	path := statePath(t)
	if err := state.NewStore(path).Save(state.State{RepoPath: "/a"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^repo_path = ["']/a["']\n?$`).Match(data) {
		t.Fatalf("file = %q, want a single repo_path line", data)
	}
}

func TestStore_SaveReplacesAtomicallyLeavingNoTempFiles(t *testing.T) {
	path := statePath(t)
	s := state.NewStore(path)
	for _, repo := range []string{"/a", "/b"} {
		if err := s.Save(state.State{RepoPath: repo}); err != nil {
			t.Fatalf("Save(%s): %v", repo, err)
		}
	}
	got, _ := state.NewStore(path).Load()
	if got.RepoPath != "/b" {
		t.Errorf("RepoPath = %q, want the second save to replace the first", got.RepoPath)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.toml" {
		t.Errorf("state dir entries = %v, want only state.toml", entries)
	}
}

func TestStore_CorruptFileWarnsAndStaysUntouched(t *testing.T) {
	path := statePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const corrupt = "repo_path = [not toml\n"
	if err := os.WriteFile(path, []byte(corrupt), 0o644); err != nil {
		t.Fatal(err)
	}
	got, warning := state.NewStore(path).Load()
	if got != (state.State{}) {
		t.Errorf("state = %+v, want defaults", got)
	}
	if !strings.Contains(warning, path) || !strings.Contains(warning, "fix or delete it") {
		t.Errorf("warning = %q, want it to name %q and say \"fix or delete it\"", warning, path)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != corrupt {
		t.Errorf("file = %q, %v; want bytes untouched", data, err)
	}
}

func TestStore_SaveReplacesCorruptFile(t *testing.T) {
	path := statePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("repo_path = [not toml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := state.NewStore(path)
	if err := s.Save(state.State{RepoPath: "/new"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, warning := s.Load()
	if got.RepoPath != "/new" || warning != "" {
		t.Fatalf("Load = %+v, %q; want RepoPath /new and no warning", got, warning)
	}
}

func TestDiscard_SaveWritesNothing(t *testing.T) {
	root := t.TempDir()
	d := state.Discard(state.NewStore(filepath.Join(root, "nuytek-harness", "state.toml")))
	before := testutil.TreeHash(t, root)
	if err := d.Save(state.State{RepoPath: "/a"}); err != nil {
		t.Fatalf("Save = %v, want nil", err)
	}
	if after := testutil.TreeHash(t, root); after != before {
		t.Fatal("dry-run Save changed the state directory")
	}
}
