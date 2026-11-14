package state_test

import (
	"os"
	"path/filepath"
	"reflect"
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
	if !reflect.DeepEqual(got, state.State{}) || warning != "" {
		t.Fatalf("Load = %+v, %q; want empty state and no warning", got, warning)
	}
}

func TestStore_SavedFileIsTOMLWithOnlyTheSetRepoPath(t *testing.T) {
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
	if !reflect.DeepEqual(got, state.State{}) {
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

func TestStore_SaveMovesCorruptFileAsideAndWritesAValidOne(t *testing.T) {
	path := statePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const corrupt = "repo_path = [not toml\n"
	if err := os.WriteFile(path+".corrupt", []byte("older corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(corrupt), 0o644); err != nil {
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
	asides, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(asides) != 1 {
		t.Fatalf("timestamped backups = %v, %v; want exactly one", asides, err)
	}
	if kept, err := os.ReadFile(asides[0]); err != nil || string(kept) != corrupt {
		t.Errorf("%s = %q, %v; want the replaced bytes", asides[0], kept, err)
	}
	if older, err := os.ReadFile(path + ".corrupt"); err != nil || string(older) != "older corrupt" {
		t.Errorf(".corrupt = %q, %v; want the earlier backup untouched", older, err)
	}
}

func TestStore_SaveAbortsOnAnUnreadableFileWithoutReplacingIt(t *testing.T) {
	path := statePath(t)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := state.NewStore(path).Save(state.State{RepoPath: "/new"}); err == nil {
		t.Fatal("Save succeeded over an unreadable state file")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Errorf("state path = %v, %v; want it left untouched", info, err)
	}
}

func TestStore_SaveThenLoadRoundTripsEveryField(t *testing.T) {
	s := state.NewStore(statePath(t))
	want := state.State{
		RepoPath:    "/repo",
		SpecterRoot: "/data/specs",
		Targets:     map[string]state.TargetState{"claude": {BackupsDir: "/data/bk"}},
	}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, warning := s.Load()
	if !reflect.DeepEqual(got, want) || warning != "" {
		t.Fatalf("Load = %+v, %q; want %+v and no warning", got, warning, want)
	}
}

func TestStore_SavePreservesKeysItDoesNotKnow(t *testing.T) {
	const existing = `repo_path = "/old"
future_flag = "keep-me"

[targets.claude]
backups_dir = "/old/bk"
future_field = "keep-field"

[targets.pi]
backups_dir = "/pi/bk"
`
	path := statePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	next := state.State{RepoPath: "/new", Targets: map[string]state.TargetState{"claude": {BackupsDir: "/new/bk"}}}
	if err := state.NewStore(path).Save(next); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"keep-me", "keep-field", "/pi/bk", "/new", "/new/bk"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("file = %q, want it to contain %q", data, want)
		}
	}
	if strings.Contains(string(data), "/old/bk") {
		t.Errorf("file = %q, want the known field overwritten", data)
	}
}

func TestStore_EmptyValueRemovesItsKey(t *testing.T) {
	path := statePath(t)
	s := state.NewStore(path)
	full := state.State{
		RepoPath:    "/repo",
		SpecterRoot: "/data/specs",
		Targets:     map[string]state.TargetState{"claude": {BackupsDir: "/data/bk"}},
	}
	if err := s.Save(full); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save(state.State{RepoPath: "/repo", Targets: map[string]state.TargetState{"claude": {}}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"specter_root", "backups_dir", "/data/specs", "/data/bk"} {
		if strings.Contains(string(data), key) {
			t.Errorf("file = %q, want %q removed", data, key)
		}
	}
	if !strings.Contains(string(data), "/repo") {
		t.Errorf("file = %q, want repo_path kept", data)
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
