package claude

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

// mcpExpandedKeys are the server keys where Claude Code expands ${VAR}.
var mcpExpandedKeys = []string{"command", "args", "env", "url", "headers"}

type mcpServers map[string]map[string]any

// readMCPSource reads the harness-owned server definitions at the repo-relative source (design D6).
func readMCPSource(env check.Env, source string) (mcpServers, error) {
	var servers mcpServers
	err := readJSON(env.FS, filepath.Join(env.Repo, source), &servers)
	return servers, err
}

func mcpEnvTexts(source string) func(check.Env) ([]check.TextSource, error) {
	return func(env check.Env) ([]check.TextSource, error) { return mcpTexts(env, source) }
}

func mcpCommandRequirements(source string) func(check.Env) ([]check.Requirement, error) {
	return func(env check.Env) ([]check.Requirement, error) { return mcpCommands(env, source) }
}

func mcpTexts(env check.Env, source string) ([]check.TextSource, error) {
	servers, err := readMCPSource(env, source)
	if err != nil {
		return nil, err
	}
	var sources []check.TextSource
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		var leaves []string
		for _, key := range mcpExpandedKeys {
			leaves = appendStringLeaves(leaves, servers[name][key])
		}
		sources = append(sources, check.TextSource{Subject: name, Text: strings.Join(leaves, "\n")})
	}
	return sources, nil
}

func appendStringLeaves(leaves []string, value any) []string {
	switch v := value.(type) {
	case string:
		return append(leaves, v)
	case []any:
		for _, item := range v {
			leaves = appendStringLeaves(leaves, item)
		}
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			leaves = appendStringLeaves(leaves, v[key])
		}
	}
	return leaves
}

func mcpCommands(env check.Env, source string) ([]check.Requirement, error) {
	servers, err := readMCPSource(env, source)
	if err != nil {
		return nil, err
	}
	var reqs []check.Requirement
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		if command, _ := servers[name]["command"].(string); command != "" {
			reqs = append(reqs, check.Requirement{Subject: name, Command: command})
		}
	}
	return reqs, nil
}

func evaluateMCP(_ context.Context, env check.Env, c manifest.Component) drift.Status {
	status := func(state drift.State, reason string) drift.Status {
		return drift.Status{Name: c.Name, State: state, Reason: reason}
	}

	owned, err := readMCPSource(env, c.Source)
	if err != nil {
		return status(drift.StateUnknown, err.Error())
	}
	var deployed struct {
		MCPServers mcpServers `json:"mcpServers"`
	}
	if err := readJSON(env.FS, filepath.Join(env.Home, c.Target), &deployed); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return status(drift.StateNotInstalled, "")
		}
		return status(drift.StateUnknown, err.Error())
	}

	var differing []string
	present := false
	for _, name := range slices.Sorted(maps.Keys(owned)) {
		got, ok := deployed.MCPServers[name]
		present = present || ok
		if !ok || !reflect.DeepEqual(got, owned[name]) {
			differing = append(differing, name)
		}
	}
	switch {
	case !present:
		return status(drift.StateNotInstalled, "")
	case len(differing) > 0:
		return status(drift.StateDrift, "differs or missing: "+strings.Join(differing, ","))
	}
	return status(drift.StateInstalled, "")
}
