package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const storeListCommand = "openspec store list --json"

func TestLoadRegistration_ListsRegisteredStoresAndFolders(t *testing.T) {
	root := t.TempDir()
	writeStore(t, root, "alpha", "id: alpha\n")
	writeStore(t, root, "beta", "name: no id here\n")
	exec := &fakeExec{
		onPath: map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{
			storeListCommand: {Stdout: []byte(`{"stores":[{"id":"alpha","root":"` + filepath.Join(root, "alpha") + `"}],"status":[]}`)},
		},
	}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if !reg.RootExists || !reg.Openspec.Found {
		t.Fatalf("RootExists=%v Openspec.Found=%v, want both true", reg.RootExists, reg.Openspec.Found)
	}
	if len(reg.Registered) != 1 || reg.Registered[0].ID != "alpha" || reg.Registered[0].Root != filepath.Join(root, "alpha") {
		t.Errorf("Registered = %+v, want alpha at its folder", reg.Registered)
	}
	want := map[string]string{filepath.Join(root, "alpha"): "alpha", filepath.Join(root, "beta"): ""}
	if len(reg.Folders) != len(want) {
		t.Fatalf("Folders = %+v, want %d entries", reg.Folders, len(want))
	}
	for _, f := range reg.Folders {
		if id, ok := want[f.Dir]; !ok || id != f.ID {
			t.Errorf("folder %+v, want id %q (empty = no id in store.yaml)", f, want[f.Dir])
		}
	}
	if got := exec.callsTo("openspec"); len(got) != 1 {
		t.Errorf("openspec calls = %v, want exactly one store list", got)
	}
}

func TestLoadRegistration_OpenspecMissingSkipsCallAndMarksRegisteredUnknown(t *testing.T) {
	root := t.TempDir()
	writeStore(t, root, "alpha", "id: alpha\n")
	exec := &fakeExec{}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if reg.Openspec.Found {
		t.Error("Openspec.Found = true, want false")
	}
	if reg.Registered != nil || reg.RegisteredErr == nil {
		t.Errorf("Registered=%v RegisteredErr=%v, want nil list and a reason", reg.Registered, reg.RegisteredErr)
	}
	if got := exec.callsTo("openspec"); len(got) != 0 {
		t.Errorf("openspec was run: %v", got)
	}
	if len(reg.Folders) != 1 {
		t.Errorf("Folders = %+v, want folders still listed", reg.Folders)
	}
}

func TestLoadRegistration_RootMissing(t *testing.T) {
	tests := []struct{ name, root string }{
		{"absent directory", filepath.Join(t.TempDir(), "nope")},
		{"relative env value", "relative/specter"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := &fakeExec{
				onPath:  map[string]bool{"openspec": true},
				results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[],"status":[]}`)}},
			}

			reg := specter.LoadRegistration(context.Background(), envWith(t, exec, tc.root))

			if reg.RootExists || len(reg.Folders) != 0 {
				t.Errorf("RootExists=%v Folders=%v, want false and none", reg.RootExists, reg.Folders)
			}
			if !reg.Openspec.Found {
				t.Error("Openspec.Found = false, want true: root missing must not hide the tool")
			}
		})
	}
}

func TestLoadRegistration_NeverRunsGit(t *testing.T) {
	exec := &fakeExec{
		onPath:  map[string]bool{"openspec": true, "git": true},
		results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[],"status":[]}`)}},
	}
	missingRoot := filepath.Join(t.TempDir(), "nope")

	specter.LoadRegistration(context.Background(), envWith(t, exec, missingRoot))

	if got := exec.callsTo("git"); len(got) != 0 {
		t.Errorf("git was run: %v", got)
	}
	if got := exec.callsTo("lookpath git"); len(got) != 0 {
		t.Errorf("git was looked up: %v", got)
	}
}

func TestLoadRegistration_UnreadableRootIsUnknownNotMissing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	exec := &fakeExec{
		onPath:  map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[],"status":[]}`)}},
	}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if reg.FoldersErr == nil || reg.RootExists {
		t.Errorf("FoldersErr=%v RootExists=%v, want an error and no claim that the root exists", reg.FoldersErr, reg.RootExists)
	}
}

func TestLoadRegistration_RelativeRootIsFlagged(t *testing.T) {
	exec := &fakeExec{
		onPath:  map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[],"status":[]}`)}},
	}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, "relative/specter"))

	if !reg.Root.Relative || reg.Root.OutsideHome || reg.FoldersErr != nil {
		t.Errorf("Root=%+v FoldersErr=%v, want relative, not outside home, no error", reg.Root, reg.FoldersErr)
	}
}

func TestLoadRegistration_StoreYAMLEdgeCases(t *testing.T) {
	root := t.TempDir()
	writeStore(t, root, "garbled", "id: [unclosed\n  - : :")
	writeStore(t, root, "huge", "id: huge\n#"+strings.Repeat("x", 64<<10))
	writeFile(t, filepath.Join(root, "stray-file"), "not a store folder")
	exec := &fakeExec{
		onPath:  map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[],"status":[]}`)}},
	}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if len(reg.Folders) != 1 || reg.Folders[0].Dir != filepath.Join(root, "garbled") || reg.Folders[0].ID != "" {
		t.Errorf("Folders = %+v, want only garbled with an empty id (oversized store.yaml and files skipped)", reg.Folders)
	}
	if reg.FoldersErr != nil {
		t.Errorf("FoldersErr = %v, want nil", reg.FoldersErr)
	}
}

func TestLoadRegistration_SymlinkedStoreYAMLIsRead(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(t.TempDir(), "real-store.yaml")
	writeFile(t, real, "id: linked\n")
	link := filepath.Join(root, "linked", ".openspec-store", "store.yaml")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: map[string]check.ExecResult{storeListCommand: {Stdout: []byte(`{"stores":[]}`)}}}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if len(reg.Folders) != 1 || reg.Folders[0].ID != "linked" || reg.FoldersErr != nil {
		t.Errorf("Folders=%+v FoldersErr=%v, want the symlinked store.yaml id read", reg.Folders, reg.FoldersErr)
	}
}

func TestLoadRegistration_StoreListFailureKeepsFolders(t *testing.T) {
	root := t.TempDir()
	writeStore(t, root, "alpha", "id: alpha\n")
	exec := &fakeExec{
		onPath:  map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{storeListCommand: {ExitCode: 1, Stderr: []byte("registry broken")}},
	}

	reg := specter.LoadRegistration(context.Background(), envWith(t, exec, root))

	if reg.RegisteredErr == nil || reg.Registered != nil {
		t.Errorf("RegisteredErr=%v Registered=%v, want an error and nil list", reg.RegisteredErr, reg.Registered)
	}
	if len(reg.Folders) != 1 {
		t.Errorf("Folders = %+v, want folders kept", reg.Folders)
	}
}
