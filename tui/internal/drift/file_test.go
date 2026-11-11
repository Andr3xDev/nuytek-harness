package drift_test

import (
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestFileEvaluatorReasons(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f fixture)
		want  string
	}{
		{"target symlink", func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o644)
			f.writeTarget("elsewhere/a.cfg", "same", 0o644)
			f.symlink(f.home, ".tooldir/a.cfg", f.home+"/elsewhere/a.cfg")
		}, "target is a symlink"},
		{"target directory", func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o644)
			f.writeTarget(".tooldir/a.cfg/inner", "x", 0o644)
		}, "target is not a regular file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			got := f.evaluate(manifest.KindFile, "src/a.cfg", ".tooldir/a.cfg")

			requireState(t, got, drift.StateDrift)
			if got.Reason != tt.want {
				t.Errorf("reason = %q, want %q", got.Reason, tt.want)
			}
		})
	}
}

func TestFileEvaluator(t *testing.T) {
	tests := []struct {
		name  string
		kind  manifest.Kind
		setup func(f fixture)
		want  drift.State
	}{
		{"identical", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o644)
			f.writeTarget(".tooldir/a.cfg", "same", 0o644)
		}, drift.StateInstalled},
		{"changed with same size", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "abc", 0o644)
			f.writeTarget(".tooldir/a.cfg", "abd", 0o644)
		}, drift.StateDrift},
		{"changed size", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "abc", 0o644)
			f.writeTarget(".tooldir/a.cfg", "abcdef", 0o644)
		}, drift.StateDrift},
		{"target absent", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "abc", 0o644)
		}, drift.StateNotInstalled},
		{"source missing is unknown", manifest.KindFile, func(f fixture) {
			f.writeTarget(".tooldir/a.cfg", "abc", 0o644)
		}, drift.StateUnknown},
		{"target symlink is drift", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o644)
			f.writeTarget("elsewhere/a.cfg", "same", 0o644)
			f.symlink(f.home, ".tooldir/a.cfg", f.home+"/elsewhere/a.cfg")
		}, drift.StateDrift},
		{"source symlink with the same content is installed", manifest.KindFile, func(f fixture) {
			f.writeSource("real/a.cfg", "same", 0o644)
			f.symlink(f.repo, "src/a.cfg", f.repo+"/real/a.cfg")
			f.writeTarget(".tooldir/a.cfg", "same", 0o644)
		}, drift.StateInstalled},
		{"source symlink with different content is drift", manifest.KindFile, func(f fixture) {
			f.writeSource("real/a.cfg", "same", 0o644)
			f.symlink(f.repo, "src/a.cfg", f.repo+"/real/a.cfg")
			f.writeTarget(".tooldir/a.cfg", "other", 0o644)
		}, drift.StateDrift},
		{"filex without exec bit is drift", manifest.KindFileX, func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o755)
			f.writeTarget(".tooldir/a.cfg", "same", 0o644)
		}, drift.StateDrift},
		{"filex with exec bit is installed", manifest.KindFileX, func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o755)
			f.writeTarget(".tooldir/a.cfg", "same", 0o755)
		}, drift.StateInstalled},
		{"file ignores exec bit", manifest.KindFile, func(f fixture) {
			f.writeSource("src/a.cfg", "same", 0o755)
			f.writeTarget(".tooldir/a.cfg", "same", 0o644)
		}, drift.StateInstalled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			got := f.evaluate(tt.kind, "src/a.cfg", ".tooldir/a.cfg")

			requireState(t, got, tt.want)
			if got.Name != "example" {
				t.Errorf("Name = %q, want example", got.Name)
			}
		})
	}
}
