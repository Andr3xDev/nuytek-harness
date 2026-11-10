package action

import (
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func componentPaths(env check.Env, c manifest.Component) (src, dst string) {
	return filepath.Join(env.Repo, c.Source), filepath.Join(env.Home, c.Target)
}

func changePaths(rows []plan.Change) []string {
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}
	return paths
}
