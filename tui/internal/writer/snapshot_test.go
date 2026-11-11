package writer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestSnapshot(t *testing.T) {
	tests := []struct {
		name      string
		files     fixture
		exec      string // relative path set to 0755 before the run
		paths     []string
		globs     []string
		outside   bool        // append a path outside Base to Paths
		dirMode   os.FileMode // when set, applied to the "d" source directory before the run
		lookalike bool        // append "<base>-x/f" to Paths
		want      fixture
		wantFail  bool
	}{
		{name: "one file", files: fixture{"a/f.txt": "A"}, paths: []string{"a/f.txt"}, want: fixture{"a/f.txt": "A"}},
		{
			name:  "whole directory",
			files: fixture{"d/x": "X", "d/sub/y": "Y"}, paths: []string{"d"},
			want: fixture{"d/x": "X", "d/sub/y": "Y"},
		},
		{
			name:  "glob expanded at apply time",
			files: fixture{"d/a.md": "A", "d/b.md": "B", "d/c.txt": "C"}, globs: []string{"d/*.md"},
			want: fixture{"d/a.md": "A", "d/b.md": "B"},
		},
		{
			name:  "first copy wins on a duplicate path",
			files: fixture{"d/a": "A"}, paths: []string{"d", "d/a", "d"}, globs: []string{"d/*"},
			want: fixture{"d/a": "A"},
		},
		{
			name:  "symlink stored as symlink",
			files: fixture{"real": "R", "ln": "->real"}, paths: []string{"ln"},
			want: fixture{"ln": "->real"},
		},
		{name: "executable keeps its mode", files: fixture{"h/run.sh": "#!"}, exec: "h/run.sh", paths: []string{"h/run.sh"}, want: fixture{"h/run.sh": "#!"}},
		{name: "missing path is skipped", files: fixture{"a": "A"}, paths: []string{"a", "gone.json"}, want: fixture{"a": "A"}},
		{name: "glob without match is skipped", files: fixture{"a": "A"}, paths: []string{"a"}, globs: []string{"zzz/*"}, want: fixture{"a": "A"}},
		{name: "path outside Base fails with nothing written", files: fixture{"a": "A"}, paths: []string{"a"}, outside: true, wantFail: true},
		{name: "look-alike path fails as outside Base", files: fixture{"a": "A"}, paths: []string{"a"}, lookalike: true, wantFail: true},
		{name: "glob escaping Base through .. fails", files: fixture{"a": "A"}, paths: []string{"a"}, globs: []string{"../*.txt"}, wantFail: true},
		{name: "source directory mode 0750 is kept", files: fixture{"d/x": "X"}, dirMode: 0o750, paths: []string{"d"}, want: fixture{"d/x": "X"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			base, storePath := filepath.Join(root, "home"), filepath.Join(root, "store")
			outsideFile := filepath.Join(root, "outside.txt")
			build(t, base, tc.files)
			build(t, root, fixture{"outside.txt": "O"})
			if tc.exec != "" {
				setMode(t, filepath.Join(base, tc.exec), 0o755)
			}
			if tc.dirMode != 0 {
				setMode(t, filepath.Join(base, "d"), tc.dirMode)
			}
			step := plan.Snapshot{Store: backup.Dir{Path: storePath, Base: base}}
			if tc.lookalike {
				build(t, root, fixture{"home-x/f": "L"})
				step.Paths = append(step.Paths, filepath.Join(base+"-x", "f"))
			}
			for _, p := range tc.paths {
				step.Paths = append(step.Paths, filepath.Join(base, p))
			}
			if tc.outside {
				step.Paths = append(step.Paths, outsideFile)
			}
			for _, g := range tc.globs {
				step.Globs = append(step.Globs, filepath.Join(base, g))
			}

			res, _ := run(t, newWriter(fixedNow), step)

			if tc.wantFail {
				wantStatus(t, res, plan.StepFailed)
				if res.Err == nil {
					t.Fatal("want an error")
				}
				equalTree(t, treeOf(t, storePath), fixture{})
				return
			}
			wantStatus(t, res, plan.StepDone)
			stamp := backup.NewStamp(backup.StoreHome, fixedNow, func(string) bool { return false })
			if res.Detail != stamp {
				t.Fatalf("Detail = %q, want %q", res.Detail, stamp)
			}
			equalTree(t, treeOf(t, filepath.Join(storePath, res.Detail)), tc.want)
			if tc.dirMode != 0 {
				if got := modeOf(t, filepath.Join(storePath, res.Detail, "d")); got != tc.dirMode {
					t.Fatalf("dir mode = %o, want %o", got, tc.dirMode)
				}
			}
			if tc.exec != "" {
				if got := modeOf(t, filepath.Join(storePath, res.Detail, tc.exec)); got != 0o755 {
					t.Fatalf("mode = %o, want 755", got)
				}
			}
		})
	}
}

func TestSnapshotNothingExistsCreatesNoStore(t *testing.T) {
	root := t.TempDir()
	base, storePath := filepath.Join(root, "home"), filepath.Join(root, "store")
	build(t, base, fixture{})
	step := plan.Snapshot{
		Store: backup.Dir{Path: storePath, Base: base},
		Paths: []string{filepath.Join(base, "gone.json")},
		Globs: []string{filepath.Join(base, "zzz/*")},
	}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepDone)
	if res.Detail != "" {
		t.Fatalf("Detail = %q, want empty", res.Detail)
	}
	if _, err := os.Lstat(storePath); !os.IsNotExist(err) {
		t.Fatalf("store dir exists after a snapshot of nothing: %v", err)
	}
}

func TestSnapshotRepoStoreMode(t *testing.T) {
	root := t.TempDir()
	base, storePath := filepath.Join(root, "repo"), filepath.Join(root, "store")
	build(t, base, fixture{"plain": "P", "run.sh": "#!"})
	setMode(t, filepath.Join(base, "run.sh"), 0o755)
	must(t, os.MkdirAll(storePath, 0o755))
	// A non-UTC zone proves the repo stamp is formatted in UTC: 23:30 at +05:00 is 18:30Z.
	now := time.Date(2026, 11, 10, 23, 30, 0, 0, time.FixedZone("plus5", 5*60*60))
	step := plan.Snapshot{
		Store: backup.Dir{Path: storePath, Base: base, Kind: backup.StoreRepo},
		Paths: []string{filepath.Join(base, "plain"), filepath.Join(base, "run.sh")},
	}

	res, _ := run(t, newWriter(now), step)

	wantStatus(t, res, plan.StepDone)
	if res.Detail != "20261110T183000Z" {
		t.Fatalf("Detail = %q, want 20261110T183000Z", res.Detail)
	}
	stampDir := filepath.Join(storePath, res.Detail)
	if got := modeOf(t, stampDir); got != 0o700 {
		t.Fatalf("stamp dir mode = %o, want 700", got)
	}
	for name, want := range map[string]os.FileMode{"plain": 0o600, "run.sh": 0o700} {
		if got := modeOf(t, filepath.Join(stampDir, name)); got != want {
			t.Fatalf("%s mode = %o, want %o", name, got, want)
		}
	}
}

func TestSnapshotCollision(t *testing.T) {
	root := t.TempDir()
	base, storePath := filepath.Join(root, "repo"), filepath.Join(root, "store")
	build(t, base, fixture{"a": "A"})
	must(t, os.MkdirAll(filepath.Join(storePath, "20261110T120000Z"), 0o700))
	step := plan.Snapshot{
		Store: backup.Dir{Path: storePath, Base: base, Kind: backup.StoreRepo},
		Paths: []string{filepath.Join(base, "a")},
	}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepDone)
	if res.Detail != "20261110T120000Z-1" {
		t.Fatalf("Detail = %q, want 20261110T120000Z-1", res.Detail)
	}
	equalTree(t, treeOf(t, filepath.Join(storePath, res.Detail)), fixture{"a": "A"})
}

func TestSnapshotMidCopyFailureLeavesNoStamp(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads files regardless of mode")
	}
	root := t.TempDir()
	base, storePath := filepath.Join(root, "home"), filepath.Join(root, "store")
	build(t, base, fixture{"a": "A", "b": "B"})
	secret := filepath.Join(base, "b")
	setMode(t, secret, 0o000)
	t.Cleanup(func() { _ = os.Chmod(secret, 0o644) })
	step := plan.Snapshot{Store: backup.Dir{Path: storePath, Base: base}, Paths: []string{filepath.Join(base, "a"), secret}}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepFailed)
	entries, err := os.ReadDir(storePath)
	must(t, err)
	if len(entries) != 0 {
		t.Fatalf("store entries = %v, want none", entries)
	}
}

func TestSnapshotAbortsWhenStoreUnwritable(t *testing.T) {
	root := t.TempDir()
	base, probe := filepath.Join(root, "home"), filepath.Join(root, "probe")
	build(t, base, fixture{"a": "A"})
	build(t, root, fixture{"probe": "P"})
	storePath := filepath.Join(probe, "store")
	step := plan.Snapshot{Store: backup.Dir{Path: storePath, Base: base}, Paths: []string{filepath.Join(base, "a")}}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepFailed)
	if res.Err == nil || !strings.Contains(res.Err.Error(), storePath) {
		t.Fatalf("err = %v, want it to name %s", res.Err, storePath)
	}
	if b, err := os.ReadFile(probe); err != nil || string(b) != "P" {
		t.Fatalf("probe = %q, %v; want untouched", b, err)
	}
}
