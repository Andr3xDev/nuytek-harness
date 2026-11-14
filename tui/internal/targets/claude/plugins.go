package claude

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

const installedPluginsPath = ".claude/plugins/installed_plugins.json"

func evaluatePlugins(_ context.Context, env check.Env, c manifest.Component) drift.Status {
	status := func(state drift.State, reason string) drift.Status {
		return drift.Status{Name: c.Name, State: state, Reason: reason}
	}

	var settings struct {
		EnabledPlugins map[string]any `json:"enabledPlugins"`
	}
	if err := readJSON(env.FS, filepath.Join(env.Home, c.Target), &settings); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return status(drift.StateNotInstalled, "")
		}
		return status(drift.StateUnknown, err.Error())
	}
	var enabled []string
	for name, value := range settings.EnabledPlugins {
		if value == true {
			enabled = append(enabled, name)
		}
	}
	if len(enabled) == 0 {
		return status(drift.StateNotInstalled, "")
	}
	slices.Sort(enabled)

	var installed struct {
		Plugins map[string]any `json:"plugins"`
	}
	if err := readJSON(env.FS, filepath.Join(env.Home, installedPluginsPath), &installed); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return status(drift.StateUnknown, err.Error())
	}
	missing := slices.DeleteFunc(slices.Clone(enabled), func(name string) bool {
		_, ok := installed.Plugins[name]
		return ok
	})
	if len(missing) > 0 {
		return status(drift.StateDrift, "missing: "+strings.Join(missing, ","))
	}
	return status(drift.StateInstalled, "")
}
