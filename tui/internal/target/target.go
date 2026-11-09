package target

import (
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

// Target.Err carries manifest and composition errors so one broken target never stops the TUI.
type Target struct {
	Spec     Spec
	Manifest manifest.Manifest
	Checks   []check.Check
	Err      error
}
