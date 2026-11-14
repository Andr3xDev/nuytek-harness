package claude

import (
	"encoding/json"
	"fmt"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// readJSON wraps with %w so callers can test for fs.ErrNotExist.
func readJSON(fsys check.FS, path string, v any) error {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
