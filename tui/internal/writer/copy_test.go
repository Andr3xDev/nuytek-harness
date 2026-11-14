package writer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestCopy(t *testing.T) {
	tests := []struct {
		name             string
		src, dst         fixture
		srcMode, dstMode map[string]os.FileMode
		srcPath, dstPath string // relative to the temp roots; empty means the root
		files            []string
		rows             []plan.Change
		want             fixture
		wantMode         map[string]os.FileMode
	}{
		{
			name: "listed files overwritten", src: fixture{"a": "new", "b": "new"}, dst: fixture{"a": "old"},
			files: []string{"a", "b"}, want: fixture{"a": "new", "b": "new"},
		},
		{
			name: "extra file next to them survives", src: fixture{"a": "new"}, dst: fixture{"a": "old", "extra": "keep"},
			files: []string{"a"}, want: fixture{"a": "new", "extra": "keep"},
		},
		{
			name: "parents created", src: fixture{"deep/x/f": "F"}, dst: fixture{},
			files: []string{"deep/x/f"}, want: fixture{"deep/x/f": "F"},
		},
		{
			name: "dot copies the single file", src: fixture{"f": "F"}, dst: fixture{},
			srcPath: "f", dstPath: "out/f", files: []string{"."}, want: fixture{"out/f": "F"},
		},
		{
			name: "source mode wins on create", src: fixture{"run.sh": "#!"}, dst: fixture{},
			srcMode: map[string]os.FileMode{"run.sh": 0o755}, files: []string{"run.sh"},
			want: fixture{"run.sh": "#!"}, wantMode: map[string]os.FileMode{"run.sh": 0o755},
		},
		{
			name: "source mode wins on overwrite", src: fixture{"run.sh": "new"}, dst: fixture{"run.sh": "old"},
			srcMode: map[string]os.FileMode{"run.sh": 0o755}, files: []string{"run.sh"},
			want: fixture{"run.sh": "new"}, wantMode: map[string]os.FileMode{"run.sh": 0o755},
		},
		{
			name: "link inside a tree recreated as a link", src: fixture{"a": "A", "ln": "->a"}, dst: fixture{},
			files: []string{"a", "ln"}, want: fixture{"a": "A", "ln": "->a"},
		},
		{
			name: "single-file source link followed", src: fixture{"real": "R", "ln": "->real"}, dst: fixture{},
			srcPath: "ln", dstPath: "out/f", files: []string{"."}, want: fixture{"out/f": "R"},
		},
		{
			name: "skip rows only with no files is a no-op", src: fixture{"a": "A"}, dst: fixture{"b": "B"},
			rows: []plan.Change{{Effect: plan.EffectSkip, Path: "a", Note: "not pulled"}}, want: fixture{"b": "B"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srcRoot, dstRoot := t.TempDir(), t.TempDir()
			build(t, srcRoot, tc.src)
			build(t, dstRoot, tc.dst)
			for rel, m := range tc.srcMode {
				setMode(t, filepath.Join(srcRoot, rel), m)
			}
			step := plan.Copy{
				Src: filepath.Join(srcRoot, tc.srcPath), Dst: filepath.Join(dstRoot, tc.dstPath),
				Files: tc.files, Rows: tc.rows,
			}

			res, _ := run(t, newWriter(fixedNow), step)

			wantStatus(t, res, plan.StepDone)
			equalTree(t, treeOf(t, dstRoot), tc.want)
			for rel, want := range tc.wantMode {
				if got := modeOf(t, filepath.Join(dstRoot, rel)); got != want {
					t.Fatalf("%s mode = %o, want %o", rel, got, want)
				}
			}
		})
	}
}

func TestCopyReplacesSymlink(t *testing.T) {
	srcRoot, dstRoot, elsewhere := t.TempDir(), t.TempDir(), t.TempDir()
	build(t, srcRoot, fixture{"a": "new"})
	build(t, elsewhere, fixture{"target": "T"})
	build(t, dstRoot, fixture{"a": "->" + filepath.Join(elsewhere, "target")})

	res, _ := run(t, newWriter(fixedNow), plan.Copy{Src: srcRoot, Dst: dstRoot, Files: []string{"a"}})

	wantStatus(t, res, plan.StepDone)
	equalTree(t, treeOf(t, dstRoot), fixture{"a": "new"})
	equalTree(t, treeOf(t, elsewhere), fixture{"target": "T"})
}
