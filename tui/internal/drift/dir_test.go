package drift_test

import (
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestDirEvaluator(t *testing.T) {
	tests := []struct {
		name       string
		kind       manifest.Kind
		setup      func(f fixture)
		want       drift.State
		wantReason string
	}{
		{"identical", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("src/sub/b.md", "b", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget(".tooldir/sub/b.md", "b", 0o644)
		}, drift.StateInstalled, ""},
		{"nested change names relative path", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("src/sub/b.md", "b1", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget(".tooldir/sub/b.md", "b2", 0o644)
		}, drift.StateDrift, "sub/b.md"},
		{"file missing in target names relative path", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("src/sub/b.md", "b", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
		}, drift.StateDrift, "sub/b.md"},
		{"target absent", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
		}, drift.StateNotInstalled, ""},
		{"source missing is unknown", manifest.KindDir, func(f fixture) {
			f.writeTarget(".tooldir/a.md", "a", 0o644)
		}, drift.StateUnknown, ""},
		{"nested symlink in source is drift", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("real/b.md", "b", 0o644)
			f.symlink(f.repo, "src/sub/b.md", f.repo+"/real/b.md")
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget(".tooldir/sub/b.md", "b", 0o644)
		}, drift.StateDrift, "sub/b.md: source is a symlink"},
		{"nested symlink in source with missing destination is missing", manifest.KindDir, func(f fixture) {
			f.writeSource("real/b.md", "b", 0o644)
			f.symlink(f.repo, "src/sub/b.md", f.repo+"/real/b.md")
			f.writeTarget(".tooldir/a.md", "a", 0o644)
		}, drift.StateDrift, "sub/b.md: missing in target"},
		{"nested target symlink to file", manifest.KindDir, func(f fixture) {
			f.writeSource("src/sub/b.md", "b", 0o644)
			f.writeTarget("elsewhere/b.md", "b", 0o644)
			f.symlink(f.home, ".tooldir/sub/b.md", f.home+"/elsewhere/b.md")
		}, drift.StateDrift, "sub/b.md: target is a symlink"},
		{"nested target directory where a file is expected", manifest.KindDir, func(f fixture) {
			f.writeSource("src/sub/b.md", "b", 0o644)
			f.writeTarget(".tooldir/sub/b.md/inner", "x", 0o644)
		}, drift.StateDrift, "sub/b.md: target is not a regular file"},
		{"top-level target symlink to directory is drift", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeTarget("elsewhere/a.md", "a", 0o644)
			f.symlink(f.home, ".tooldir", f.home+"/elsewhere")
		}, drift.StateDrift, "target is not a directory"},
		{"target is a file when directory expected", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeTarget(".tooldir", "oops", 0o644)
		}, drift.StateDrift, "target is not a directory"},
		{"symlinked subdirectory in target is drift", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("src/sub/b.md", "b", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget("elsewhere/b.md", "b", 0o644)
			f.symlink(f.home, ".tooldir/sub", f.home+"/elsewhere")
		}, drift.StateDrift, "sub is not a directory"},
		{"custom target-only files stay installed", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget(".tooldir/mine.md", "custom", 0o644)
			f.writeTarget(".tooldir/mydir/deep.md", "custom", 0o644)
		}, drift.StateInstalled, ""},
		{"bytecode ignored on both sides", manifest.KindDir, func(f fixture) {
			f.writeSource("src/a.md", "a", 0o644)
			f.writeSource("src/__pycache__/m.cpython.pyc", "src-bytes", 0o644)
			f.writeSource("src/stray.pyc", "src-stray", 0o644)
			f.writeTarget(".tooldir/a.md", "a", 0o644)
			f.writeTarget(".tooldir/__pycache__/m.cpython.pyc", "different", 0o644)
			f.writeTarget(".tooldir/other.pyc", "target-only", 0o644)
		}, drift.StateInstalled, ""},
		{"dirx top-level sh without exec bit is drift", manifest.KindDirX, func(f fixture) {
			f.writeSource("src/run.sh", "#!/bin/sh", 0o755)
			f.writeTarget(".tooldir/run.sh", "#!/bin/sh", 0o644)
		}, drift.StateDrift, "run.sh"},
		{"dirx top-level sh with exec bit is installed", manifest.KindDirX, func(f fixture) {
			f.writeSource("src/run.sh", "#!/bin/sh", 0o755)
			f.writeTarget(".tooldir/run.sh", "#!/bin/sh", 0o755)
		}, drift.StateInstalled, ""},
		{"dirx non-sh without exec bit is installed", manifest.KindDirX, func(f fixture) {
			f.writeSource("src/guard.py", "print()", 0o644)
			f.writeTarget(".tooldir/guard.py", "print()", 0o644)
		}, drift.StateInstalled, ""},
		{"dirx nested sh without exec bit is installed", manifest.KindDirX, func(f fixture) {
			f.writeSource("src/sub/run.sh", "#!/bin/sh", 0o755)
			f.writeTarget(".tooldir/sub/run.sh", "#!/bin/sh", 0o644)
		}, drift.StateInstalled, ""},
		{"dir ignores exec bit on sh", manifest.KindDir, func(f fixture) {
			f.writeSource("src/run.sh", "#!/bin/sh", 0o755)
			f.writeTarget(".tooldir/run.sh", "#!/bin/sh", 0o644)
		}, drift.StateInstalled, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			got := f.evaluate(tt.kind, "src", ".tooldir")

			requireState(t, got, tt.want)
			if !strings.Contains(got.Reason, tt.wantReason) {
				t.Errorf("reason %q lacks %q", got.Reason, tt.wantReason)
			}
		})
	}
}

func TestDirEvaluator_ReasonNeverNamesTargetOnlyFiles(t *testing.T) {
	f := newFixture(t)
	f.writeSource("src/a.md", "a1", 0o644)
	f.writeTarget(".tooldir/a.md", "a2", 0o644)
	f.writeTarget(".tooldir/mine.md", "custom", 0o644)

	got := f.evaluate(manifest.KindDir, "src", ".tooldir")

	requireState(t, got, drift.StateDrift)
	if strings.Contains(got.Reason, "mine.md") {
		t.Errorf("reason %q names a target-only file", got.Reason)
	}
}
