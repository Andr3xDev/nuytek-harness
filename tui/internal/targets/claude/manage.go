package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	noSummaryText       = "no summary available"
	localSkillTag       = "local"
	versionUnknown      = "version unknown"
	titleMCPs           = "MCPs"
	titlePlugins        = "Plugins"
	titleSkills         = "Skills"
	skillFile           = "SKILL.md"
	missingComponentFmt = "manifest has no %q component"
)

func manage(_ context.Context, env check.Env, m manifest.Manifest) []target.ManageSection {
	return []target.ManageSection{manageMCPs(env, m), managePlugins(env, m), manageSkills(env, m)}
}

func findComponent(m manifest.Manifest, match func(manifest.Component) bool) (manifest.Component, bool) {
	i := slices.IndexFunc(m.Components, match)
	if i < 0 {
		return manifest.Component{}, false
	}
	return m.Components[i], true
}

func byName(name string) func(manifest.Component) bool {
	return func(c manifest.Component) bool { return c.Name == name }
}

func unknownSection(title, reason string) target.ManageSection {
	return target.ManageSection{Title: title, Unknown: reason}
}

func manageMCPs(env check.Env, m manifest.Manifest) target.ManageSection {
	c, ok := findComponent(m, byName(componentMCP))
	if !ok {
		return unknownSection(titleMCPs, fmt.Sprintf(missingComponentFmt, componentMCP))
	}
	data, err := env.FS.ReadFile(filepath.Join(env.Home, c.Target))
	if errors.Is(err, fs.ErrNotExist) {
		return target.ManageSection{Title: titleMCPs, Empty: "no MCP configuration found"}
	}
	if err != nil {
		return unknownSection(titleMCPs, err.Error())
	}
	_, servers, err := parseServers(data)
	if err != nil {
		return unknownSection(titleMCPs, err.Error())
	}
	if len(servers) == 0 {
		return target.ManageSection{Title: titleMCPs, Empty: "no MCP servers configured"}
	}
	owned, _ := ownedServers(env.FS, env.Repo, c) // unreadable source: every server reads as external

	section := target.ManageSection{Title: titleMCPs}
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		origin := "external"
		if _, harness := owned[name]; harness {
			origin = "harness-owned"
		}
		row := target.ManageRow{Name: name}
		var def struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(servers[name], &def)
		switch {
		case def.Command == "":
			row.Detail = origin + ", no command"
		default:
			if resolved, err := env.Exec.LookPath(def.Command); err != nil {
				row.Status, row.Detail = check.StatusError, origin+", "+def.Command+" not found"
			} else {
				row.Status, row.Detail = check.StatusOK, origin+", "+resolved
			}
		}
		section.Rows = append(section.Rows, row)
	}
	return section
}

func managePlugins(env check.Env, m manifest.Manifest) target.ManageSection {
	c, ok := findComponent(m, func(c manifest.Component) bool { return c.Kind == kindPlugins })
	if !ok {
		return unknownSection(titlePlugins, fmt.Sprintf(missingComponentFmt, kindPlugins))
	}
	var settings pluginSettings
	if err := readJSON(env.FS, filepath.Join(env.Home, c.Target), &settings); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return target.ManageSection{Title: titlePlugins, Empty: "no plugins configured"}
		}
		return unknownSection(titlePlugins, err.Error())
	}
	if len(settings.EnabledPlugins) == 0 {
		return target.ManageSection{Title: titlePlugins, Empty: "no plugins configured"}
	}
	installed, readable, note := readKeys(env, installedPluginsPath, installedPluginsKey)

	section := target.ManageSection{Title: titlePlugins}
	for _, id := range slices.Sorted(maps.Keys(settings.EnabledPlugins)) {
		row := target.ManageRow{Name: id}
		record, isInstalled := installed[id]
		switch {
		case settings.EnabledPlugins[id] != true:
			row.Detail = "disabled"
		case !readable:
			row.Status, row.Detail = check.StatusUnknown, note
		case !isInstalled:
			row.Status, row.Detail = check.StatusError, "not installed"
		default:
			row.Status, row.Detail = check.StatusOK, "installed "+pluginVersion(record)
		}
		section.Rows = append(section.Rows, row)
	}
	return section
}

// pluginVersion reads the first install record's version (array form), tolerating a bare object.
func pluginVersion(record json.RawMessage) string {
	var records []struct {
		Version string `json:"version"`
	}
	var single struct {
		Version string `json:"version"`
	}
	switch {
	case json.Unmarshal(record, &records) == nil && len(records) > 0 && records[0].Version != "":
		return records[0].Version
	case json.Unmarshal(record, &single) == nil && single.Version != "":
		return single.Version
	}
	return versionUnknown
}

func manageSkills(env check.Env, m manifest.Manifest) target.ManageSection {
	c, ok := findComponent(m, byName(componentSkills))
	if !ok {
		return unknownSection(titleSkills, fmt.Sprintf(missingComponentFmt, componentSkills))
	}
	dir := filepath.Join(env.Home, c.Target)
	entries, err := env.FS.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return target.ManageSection{Title: titleSkills, Empty: "no skills installed"}
	}
	if err != nil {
		return unknownSection(titleSkills, err.Error())
	}

	// Without the harness skill set, "local" would be a guess: tag nothing.
	_, sourceErr := env.FS.ReadDir(filepath.Join(env.Repo, c.Source))
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })

	section := target.ManageSection{Title: titleSkills}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if info, err := env.FS.Stat(path); err != nil || !info.IsDir() {
			continue
		}
		row := target.ManageRow{Name: entry.Name(), Preview: noSummaryText}
		if sourceErr == nil {
			if info, err := env.FS.Stat(filepath.Join(env.Repo, c.Source, entry.Name())); err != nil || !info.IsDir() {
				row.Detail = localSkillTag
			}
		}
		if fields, err := check.ParseFrontmatter(env.FS, filepath.Join(path, skillFile)); err == nil && fields.Description != "" {
			row.Preview = fields.Description
		}
		section.Rows = append(section.Rows, row)
	}
	if len(section.Rows) == 0 {
		section.Empty = "no skills installed"
	}
	return section
}
