package claude

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const PluginCommandTimeout = 2 * time.Minute

const (
	installedPluginsPath    = ".claude/plugins/installed_plugins.json"
	knownMarketplacesPath   = ".claude/plugins/known_marketplaces.json"
	skipPluginsEnv          = "NUYTEK_SKIP_PLUGINS"
	installedPluginsKey     = "plugins"
	pluginsUnreadableNote   = "%s is unparseable; treated as empty"
	pluginsNotRemovedNote   = "undeclared plugins and marketplaces were not removed"
	marketplaceNoSourceNote = "marketplace %s has no repo or url; skipped"
	invalidPluginArgNote    = "%q is not a valid plugin argument; skipped"
)

// cavecrewAgentGlobs enforce the harness rule of never using cavecrew agents, even when a plugin reinstalls them.
var cavecrewAgentGlobs = []string{
	".claude/plugins/cache/caveman/caveman/*/agents/cavecrew-*.md",
	".claude/plugins/cache/caveman/caveman/*/plugins/caveman/agents/cavecrew-*.md",
	".claude/plugins/marketplaces/caveman/agents/cavecrew-*.md",
	".claude/plugins/marketplaces/caveman/plugins/caveman/agents/cavecrew-*.md",
}

type pluginSettings struct {
	EnabledPlugins         map[string]any `json:"enabledPlugins"`
	ExtraKnownMarketplaces map[string]struct {
		Source struct {
			Repo string `json:"repo"`
			URL  string `json:"url"`
		} `json:"source"`
	} `json:"extraKnownMarketplaces"`
}

func installPlugins(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	return planPlugins(env, c, false)
}

func hardInstallPlugins(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	return planPlugins(env, c, true)
}

func planPlugins(env check.Env, c manifest.Component, hard bool) (plan.Part, error) {
	if value, _ := env.LookupEnv(skipPluginsEnv); value == "1" {
		return plan.Part{Skipped: []plan.Skip{{Name: c.Name, Reason: skipPluginsEnv + "=1"}}}, nil
	}
	var settings pluginSettings
	if err := readJSON(env.FS, filepath.Join(env.Repo, c.Source), &settings); err != nil {
		return plan.Part{}, err
	}
	known, knownOK, knownNote := readKeys(env, knownMarketplacesPath, "")
	installed, installedOK, installedNote := readKeys(env, installedPluginsPath, installedPluginsKey)

	var part plan.Part
	for _, note := range []string{knownNote, installedNote} {
		if note != "" {
			part.Notes = append(part.Notes, note)
		}
	}
	if hard && !(knownOK && installedOK) {
		part.Notes = append(part.Notes, pluginsNotRemovedNote)
	} else if hard {
		for _, id := range undeclaredKeys(installed, settings.EnabledPlugins) {
			addPluginStep(&part, id, func() plan.Exec { return pluginRemoval("claude", "plugin", "uninstall", id) })
		}
		for _, name := range undeclaredKeys(known, settings.ExtraKnownMarketplaces) {
			addPluginStep(&part, name, func() plan.Exec { return pluginRemoval("claude", "plugin", "marketplace", "remove", name) })
		}
	}

	for _, name := range slices.Sorted(maps.Keys(settings.ExtraKnownMarketplaces)) {
		if _, present := known[name]; present {
			addPluginStep(&part, name, func() plan.Exec { return pluginExec(nil, "claude", "plugin", "marketplace", "update", name) })
			continue
		}
		source := settings.ExtraKnownMarketplaces[name].Source
		location := cmp.Or(source.Repo, source.URL)
		if location == "" {
			part.Notes = append(part.Notes, fmt.Sprintf(marketplaceNoSourceNote, name))
			continue
		}
		addPluginStep(&part, location, func() plan.Exec { return pluginExec(nil, "claude", "plugin", "marketplace", "add", location) })
	}
	for _, id := range slices.Sorted(maps.Keys(settings.EnabledPlugins)) {
		if settings.EnabledPlugins[id] != true {
			continue
		}
		if _, present := installed[id]; present {
			addPluginStep(&part, id, func() plan.Exec { return pluginExec(nil, "claude", "plugin", "update", id, "-y") })
			continue
		}
		addPluginStep(&part, id, func() plan.Exec { return pluginExec(nil, "claude", "plugin", "install", id) })
	}

	step, found, err := cavecrewRemoval(env)
	if err != nil {
		return plan.Part{}, err
	}
	if found {
		part.Steps = append(part.Steps, step)
	}
	return part, nil
}

// cavecrewRemoval is planned only when a cavecrew file exists now, so a clean install previews no deletion.
func cavecrewRemoval(env check.Env) (step plan.RemoveMatching, found bool, err error) {
	globs := make([]string, len(cavecrewAgentGlobs))
	for i, glob := range cavecrewAgentGlobs {
		globs[i] = filepath.Join(plan.EscapeGlob(env.Home), glob)
		matched, err := anyMatch(env.FS, env.Home, glob)
		if err != nil {
			return plan.RemoveMatching{}, false, err
		}
		found = found || matched
	}
	return plan.RemoveMatching{Globs: globs}, found, nil
}

// anyMatch reports whether pattern, relative to base, matches an entry; base is literal and needs no escaping.
// It walks through fsys because filepath.Glob reads the real disk and hides errors.
func anyMatch(fsys check.FS, base, pattern string) (bool, error) {
	dirs := []string{base}
	for _, segment := range strings.Split(pattern, string(filepath.Separator)) {
		var next []string
		for _, dir := range dirs {
			entries, err := fsys.ReadDir(dir)
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				continue
			}
			if err != nil {
				return false, err
			}
			for _, entry := range entries {
				if ok, _ := filepath.Match(segment, entry.Name()); ok {
					next = append(next, filepath.Join(dir, entry.Name()))
				}
			}
		}
		dirs = next
	}
	return len(dirs) > 0, nil
}

// readKeys returns the entries of the JSON object at path under home, or of the object under wrapper when set.
// A missing file is empty; an unparseable one is empty with ok false and a note.
func readKeys(env check.Env, path, wrapper string) (entries map[string]json.RawMessage, ok bool, note string) {
	full := filepath.Join(env.Home, path)
	if err := readJSON(env.FS, full, &entries); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, true, ""
		}
		return nil, false, fmt.Sprintf(pluginsUnreadableNote, full)
	}
	if wrapper == "" {
		return entries, true, ""
	}
	var inner map[string]json.RawMessage
	if raw, found := entries[wrapper]; found {
		if err := json.Unmarshal(raw, &inner); err != nil {
			return nil, false, fmt.Sprintf(pluginsUnreadableNote, full)
		}
	}
	return inner, true, ""
}

func undeclaredKeys[V, D any](have map[string]V, declared map[string]D) []string {
	var keys []string
	for key := range have {
		if _, ok := declared[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// validArg refuses empty and leading-dash values: the claude CLI would read a "-x" id or name as a flag.
func validArg(s string) bool {
	return s != "" && !strings.HasPrefix(s, "-")
}

// addPluginStep appends the step built for value, or a note when value is not a safe positional argument.
func addPluginStep(part *plan.Part, value string, build func() plan.Exec) {
	if !validArg(value) {
		part.Notes = append(part.Notes, fmt.Sprintf(invalidPluginArgNote, value))
		return
	}
	part.Steps = append(part.Steps, build())
}

func pluginExec(rows []plan.Change, argv ...string) plan.Exec {
	return plan.Exec{Argv: argv, Timeout: PluginCommandTimeout, Warn: true, Rows: rows}
}

// pluginRemoval is not snapshotted, so its row is unrestorable.
func pluginRemoval(argv ...string) plan.Exec {
	row := plan.Change{Effect: plan.EffectDelete, Path: strings.Join(argv, " "), Unrestorable: true}
	return pluginExec([]plan.Change{row}, argv...)
}

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
