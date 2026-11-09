package drift

import (
	"maps"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

// Evaluators returns the data-kind evaluators merged with extra.
func Evaluators(extra map[manifest.Kind]Evaluator) map[manifest.Kind]Evaluator {
	evaluators := map[manifest.Kind]Evaluator{
		manifest.KindDir:   evaluateDir,
		manifest.KindDirX:  evaluateDir,
		manifest.KindFile:  evaluateFile,
		manifest.KindFileX: evaluateFile,
	}
	maps.Copy(evaluators, extra)
	return evaluators
}
