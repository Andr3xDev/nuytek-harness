package target

import (
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

type Spec struct {
	ID, Name   string
	Evaluators map[manifest.Kind]drift.Evaluator              // keys = kinds the manifest may use
	Checks     func(manifest.Manifest) ([]check.Check, error) // nil = planned target, not available
	BackupDirs func(home, repo string) []backup.Dir           // nil = none
}

// Planned returns a spec for a target that is announced but not implemented.
func Planned(id, name string) Spec {
	return Spec{ID: id, Name: name}
}
