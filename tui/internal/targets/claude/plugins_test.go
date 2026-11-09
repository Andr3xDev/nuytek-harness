package claude

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

const (
	pluginsSettingsRel  = ".claude/settings.json"
	installedPluginsRel = ".claude/plugins/installed_plugins.json"
)

func TestEvaluatePlugins(t *testing.T) {
	const enabled = `{"enabledPlugins": {"zeta@zeta": true, "engram@engram": true, "alpha@alpha": true}}`
	tests := []struct {
		name       string
		settings   *string
		installed  *string
		want       drift.State
		wantReason string
	}{
		{"all installed", ptr(enabled), ptr(`{"version": 2, "plugins": {"zeta@zeta": [], "engram@engram": [], "alpha@alpha": [], "extra@x": []}}`), drift.StateInstalled, ""},
		{"one missing", ptr(enabled), ptr(`{"plugins": {"zeta@zeta": [], "alpha@alpha": []}}`), drift.StateDrift, "missing: engram@engram"},
		{"several missing sorted and comma joined", ptr(enabled), ptr(`{"plugins": {"zeta@zeta": []}}`), drift.StateDrift, "missing: alpha@alpha,engram@engram"},
		{"installed file unparseable", ptr(enabled), ptr(`{"plugins": `), drift.StateUnknown, ""},
		{"disabled plugin is not required", ptr(`{"enabledPlugins": {"a@x": true, "b@x": false}}`), ptr(`{"plugins": {"a@x": []}}`), drift.StateInstalled, ""},
		{"installed file absent", ptr(`{"enabledPlugins": {"b@x": true, "a@x": true, "c@x": false}}`), nil, drift.StateDrift, "missing: a@x,b@x"},
		{"settings absent", nil, ptr(`{"plugins": {}}`), drift.StateNotInstalled, ""},
		{"only disabled plugins", ptr(`{"enabledPlugins": {"b@x": false}}`), nil, drift.StateNotInstalled, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t)
			if tt.settings != nil {
				writeFile(t, env.Home, pluginsSettingsRel, *tt.settings)
			}
			if tt.installed != nil {
				writeFile(t, env.Home, installedPluginsRel, *tt.installed)
			}
			c := manifest.Component{Name: "plugins", Kind: "plugins", Source: "claude/claude-settings.json", Target: pluginsSettingsRel}

			got := evaluatePlugins(context.Background(), env, c)

			requireState(t, got, tt.want)
			if got.Name != "plugins" {
				t.Errorf("Name = %q, want plugins", got.Name)
			}
			if tt.wantReason != "" && got.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if tt.want == drift.StateUnknown && got.Reason == "" {
				t.Error("unknown state must carry the parse reason")
			}
		})
	}
}
