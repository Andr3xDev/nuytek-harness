package target_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func okCheck(id string) check.Check {
	return check.Check{ID: id, Run: func(context.Context, check.Env) []check.Result {
		return []check.Result{{CheckID: id, Subject: id, Status: check.StatusOK}}
	}}
}

func mkManifests(t *testing.T, names ...string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(repo, "targets", n+".toml"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func implementedSpec(id string, checks func(manifest.Manifest) ([]check.Check, error)) target.Spec {
	return target.Spec{ID: id, Name: id, Checks: checks,
		Evaluators: map[manifest.Kind]drift.Evaluator{manifest.KindFile: nil}}
}

func noChecks(manifest.Manifest) ([]check.Check, error) { return nil, nil }
